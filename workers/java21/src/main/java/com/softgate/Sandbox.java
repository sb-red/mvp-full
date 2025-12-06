package com.softgate;

import com.google.gson.Gson;

import java.io.*;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Map;
import java.util.concurrent.TimeUnit;

public class Sandbox {
    private static final Gson gson = new Gson();

    public static ExecutionResult runCode(String code, Map<String, Object> inputData) {
        String status = "SUCCESS";
        String output = "";
        StringBuilder logs = new StringBuilder();
        Path tempDir = null;

        try {
            // Create temporary directory
            tempDir = Files.createTempDirectory("java_exec_");

            // Create wrapper code
            String wrapperCode = String.format("""
                import java.util.*;
                import com.google.gson.*;

                %s

                public class Main {
                    public static void main(String[] args) {
                        try {
                            String inputJson = args.length > 0 ? args[0] : "{}";
                            Gson gson = new Gson();
                            Map<String, Object> event = gson.fromJson(inputJson, Map.class);

                            Object result = Handler.handle(event);

                            String outputStr = gson.toJson(result);
                            System.out.println(outputStr);
                        } catch (Exception e) {
                            System.err.println("Error: " + e.getMessage());
                            e.printStackTrace();
                            System.exit(1);
                        }
                    }
                }
                """, code);

            // Write Main.java
            Path mainFile = tempDir.resolve("Main.java");
            Files.writeString(mainFile, wrapperCode);

            // Download Gson JAR (in production, this should be pre-downloaded)
            Path gsonJar = tempDir.resolve("gson.jar");
            downloadGson(gsonJar);

            // Compile
            ProcessBuilder compileBuilder = new ProcessBuilder(
                "javac", "-cp", gsonJar.toString(), mainFile.toString()
            );
            compileBuilder.directory(tempDir.toFile());
            compileBuilder.redirectErrorStream(true);

            Process compileProcess = compileBuilder.start();
            String compileOutput = readStream(compileProcess.getInputStream());
            boolean compiled = compileProcess.waitFor(30, TimeUnit.SECONDS);

            if (!compiled || compileProcess.exitValue() != 0) {
                status = "ERROR";
                output = "Compilation error";
                logs.append(compileOutput);
                return new ExecutionResult(status, output, logs.toString());
            }

            // Execute
            String inputJson = gson.toJson(inputData);
            ProcessBuilder execBuilder = new ProcessBuilder(
                "java", "-cp", tempDir.toString() + ":" + gsonJar.toString(), "Main", inputJson
            );
            execBuilder.directory(tempDir.toFile());

            Process execProcess = execBuilder.start();

            // Capture stdout and stderr
            String stdout = readStream(execProcess.getInputStream());
            String stderr = readStream(execProcess.getErrorStream());

            boolean finished = execProcess.waitFor(30, TimeUnit.SECONDS);

            if (!finished) {
                execProcess.destroyForcibly();
                status = "TIMEOUT";
                output = "Execution timed out (30 seconds)";
            } else if (execProcess.exitValue() != 0) {
                status = "ERROR";
                output = "Execution error";
                logs.append(stderr);
            } else {
                output = stdout.trim();
                logs.append(stderr);
            }

        } catch (Exception e) {
            status = "ERROR";
            output = e.getMessage();
            StringWriter sw = new StringWriter();
            e.printStackTrace(new PrintWriter(sw));
            logs.append(sw.toString());
        } finally {
            // Clean up
            if (tempDir != null) {
                deleteDirectory(tempDir.toFile());
            }
        }

        return new ExecutionResult(status, output, logs.toString());
    }

    private static void downloadGson(Path gsonJar) throws IOException, InterruptedException {
        ProcessBuilder pb = new ProcessBuilder(
            "wget", "-q", "-O", gsonJar.toString(),
            "https://repo1.maven.org/maven2/com/google/code/gson/gson/2.10.1/gson-2.10.1.jar"
        );
        Process process = pb.start();
        process.waitFor(30, TimeUnit.SECONDS);
    }

    private static String readStream(InputStream is) throws IOException {
        try (BufferedReader reader = new BufferedReader(new InputStreamReader(is))) {
            StringBuilder sb = new StringBuilder();
            String line;
            while ((line = reader.readLine()) != null) {
                sb.append(line).append("\n");
            }
            return sb.toString();
        }
    }

    private static void deleteDirectory(File dir) {
        if (dir.exists()) {
            File[] files = dir.listFiles();
            if (files != null) {
                for (File file : files) {
                    if (file.isDirectory()) {
                        deleteDirectory(file);
                    } else {
                        file.delete();
                    }
                }
            }
            dir.delete();
        }
    }
}
