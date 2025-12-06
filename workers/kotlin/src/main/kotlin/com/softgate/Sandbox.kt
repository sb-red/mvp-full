package com.softgate

import com.google.gson.Gson
import java.io.File
import java.nio.file.Files
import java.util.concurrent.TimeUnit

data class ExecutionResult(
    val status: String,
    val output: String,
    val logs: String
)

private val gsonJarPath: String = System.getenv("GSON_JAR_PATH") ?: "/opt/libs/gson-2.10.1.jar"

fun runCode(code: String, inputData: Map<String, Any>): ExecutionResult {
    var status = "SUCCESS"
    var output = ""
    val logs = StringBuilder()
    var tempDir: File? = null

    try {
        // Create temporary directory
        tempDir = Files.createTempDirectory("kotlin_exec_").toFile()

        // Create wrapper code
        val wrapperCode = """
import com.google.gson.*

$code

fun main(args: Array<String>) {
    try {
        val inputJson = if (args.isNotEmpty()) args[0] else "{}"
        val gson = Gson()
        val event = gson.fromJson(inputJson, Map::class.java) as Map<String, Any>

        val result = Handler.handle(event)

        val outputStr = gson.toJson(result)
        println(outputStr)
    } catch (e: Exception) {
        System.err.println("Error: ${'$'}{e.message}")
        e.printStackTrace()
        System.exit(1)
    }
}
"""

        // Write Main.kt
        val mainFile = File(tempDir, "Main.kt")
        mainFile.writeText(wrapperCode)

        // Compile Kotlin code
        val compileProcess = ProcessBuilder(
            "kotlinc",
            "-cp", gsonJarPath,
            "-include-runtime",
            "-d", "program.jar",
            mainFile.absolutePath
        ).directory(tempDir)
            .redirectErrorStream(true)
            .start()

        val compileOutput = compileProcess.inputStream.bufferedReader().readText()
        val compiled = compileProcess.waitFor(60, TimeUnit.SECONDS)

        if (!compiled || compileProcess.exitValue() != 0) {
            status = "ERROR"
            output = "Compilation error"
            logs.append(compileOutput)
            return ExecutionResult(status, output, logs.toString())
        }

        // Execute Kotlin program
        val gson = Gson()
        val inputJson = gson.toJson(inputData)

        val execProcess = ProcessBuilder(
            "java",
            "-cp", "${tempDir.absolutePath}/program.jar:$gsonJarPath",
            "MainKt",
            inputJson
        ).directory(tempDir)
            .start()

        val stdout = execProcess.inputStream.bufferedReader().readText()
        val stderr = execProcess.errorStream.bufferedReader().readText()

        val finished = execProcess.waitFor(30, TimeUnit.SECONDS)

        if (!finished) {
            execProcess.destroyForcibly()
            status = "TIMEOUT"
            output = "Execution timed out (30 seconds)"
        } else if (execProcess.exitValue() != 0) {
            status = "ERROR"
            output = "Execution error"
            logs.append(stderr)
        } else {
            output = stdout.trim()
            logs.append(stderr)
        }

    } catch (e: Exception) {
        status = "ERROR"
        output = e.message ?: "Unknown error"
        logs.append(e.stackTraceToString())
    } finally {
        // Clean up
        tempDir?.deleteRecursively()
    }

    return ExecutionResult(status, output, logs.toString())
}
