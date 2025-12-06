import Foundation

struct ExecutionResult {
    let status: String
    let output: String
    let logs: String
}

func runCode(code: String, inputData: [String: Any]) -> ExecutionResult {
    var status = "SUCCESS"
    var output = ""
    var logs = ""

    let tempDir = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)

    do {
        try FileManager.default.createDirectory(at: tempDir, withIntermediateDirectories: true)

        // Create wrapper code
        let inputJson = try JSONSerialization.data(withJSONObject: inputData).base64EncodedString()

        let wrapperCode = """
        import Foundation

        \(code)

        // Main execution
        if let inputData = Data(base64Encoded: "\(inputJson)"),
           let event = try? JSONSerialization.jsonObject(with: inputData) as? [String: Any] {
            let result = handler(event: event)

            if let resultData = try? JSONSerialization.data(withJSONObject: result),
               let resultString = String(data: resultData, encoding: .utf8) {
                print(resultString)
            }
        } else {
            fputs("Failed to parse input JSON\\n", stderr)
            exit(1)
        }
        """

        // Write main.swift
        let mainFile = tempDir.appendingPathComponent("main.swift")
        try wrapperCode.write(to: mainFile, atomically: true, encoding: .utf8)

        // Compile Swift code
        let compileProcess = Process()
        compileProcess.executableURL = URL(fileURLWithPath: "/usr/bin/swiftc")
        compileProcess.arguments = ["-o", tempDir.appendingPathComponent("program").path, mainFile.path]

        let compilePipe = Pipe()
        compileProcess.standardOutput = compilePipe
        compileProcess.standardError = compilePipe

        try compileProcess.run()
        compileProcess.waitUntilExit()

        let compileData = compilePipe.fileHandleForReading.readDataToEndOfFile()
        let compileOutput = String(data: compileData, encoding: .utf8) ?? ""

        if compileProcess.terminationStatus != 0 {
            status = "ERROR"
            output = "Compilation error"
            logs = compileOutput
            return ExecutionResult(status: status, output: output, logs: logs)
        }

        // Execute compiled program
        let execProcess = Process()
        execProcess.executableURL = tempDir.appendingPathComponent("program")

        let stdoutPipe = Pipe()
        let stderrPipe = Pipe()
        execProcess.standardOutput = stdoutPipe
        execProcess.standardError = stderrPipe

        try execProcess.run()

        // Timeout after 30 seconds
        let timeoutTask = DispatchWorkItem {
            execProcess.terminate()
        }
        DispatchQueue.global().asyncAfter(deadline: .now() + 30, execute: timeoutTask)

        execProcess.waitUntilExit()
        timeoutTask.cancel()

        let stdoutData = stdoutPipe.fileHandleForReading.readDataToEndOfFile()
        let stderrData = stderrPipe.fileHandleForReading.readDataToEndOfFile()

        let stdout = String(data: stdoutData, encoding: .utf8) ?? ""
        let stderr = String(data: stderrData, encoding: .utf8) ?? ""

        if execProcess.terminationStatus != 0 {
            if execProcess.terminationReason == .exit {
                status = "ERROR"
                output = "Execution error"
            } else {
                status = "TIMEOUT"
                output = "Execution timed out (30 seconds)"
            }
            logs = stderr
        } else {
            output = stdout.trimmingCharacters(in: .whitespacesAndNewlines)
            logs = stderr
        }

    } catch {
        status = "ERROR"
        output = error.localizedDescription
        logs = "\(error)"
    }

    // Clean up
    try? FileManager.default.removeItem(at: tempDir)

    return ExecutionResult(status: status, output: output, logs: logs)
}
