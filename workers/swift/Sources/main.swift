import Foundation
import RediStack
import NIO

let REDIS_HOST = ProcessInfo.processInfo.environment["REDIS_HOST"] ?? "localhost"
let REDIS_PORT = Int(ProcessInfo.processInfo.environment["REDIS_PORT"] ?? "6379") ?? 6379
let QUEUE_KEY = "execution_queue:swift"
let RESULT_KEY_PREFIX = "result:"
let RESULT_TTL: Int64 = 600

func main() async {
    print("Swift Worker started. Connecting to Redis at \(REDIS_HOST):\(REDIS_PORT)")

    let eventLoopGroup = MultiThreadedEventLoopGroup(numberOfThreads: 1)
    defer { try? eventLoopGroup.syncShutdownGracefully() }

    do {
        let connection = try await RedisConnection.make(
            configuration: try .init(hostname: REDIS_HOST, port: REDIS_PORT),
            boundEventLoop: eventLoopGroup.next()
        ).get()

        print("Connected to Redis successfully")

        while true {
            do {
                // Blocking pop from queue (5 second timeout)
                let result = try await connection.brpop(from: [RedisKey(QUEUE_KEY)], timeout: .seconds(5)).get()

                if let (_, rawData) = result {
                    var invocationId: Int64?

                    do {
                        guard let rawString = rawData.string,
                              let jsonData = rawString.data(using: .utf8),
                              let data = try JSONSerialization.jsonObject(with: jsonData) as? [String: Any] else {
                            print("Error: Invalid JSON format")
                            continue
                        }

                        invocationId = data["invocationId"] as? Int64 ?? (data["invocationId"] as? Int).map { Int64($0) }
                        guard let code = data["code"] as? String else {
                            print("Error: Missing code field")
                            continue
                        }
                        let inputData = data["input"] as? [String: Any] ?? [:]

                        print("Processing invocation: \(invocationId ?? -1)")

                        let startTime = Date()
                        let execResult = runCode(code: code, inputData: inputData)
                        let duration = Int(Date().timeIntervalSince(startTime) * 1000)

                        // Parse output as JSON if possible
                        var outputJson: Any? = nil
                        var errorMessage = ""
                        if execResult.status == "SUCCESS" {
                            if let outputData = execResult.output.data(using: String.Encoding.utf8),
                               let json = try? JSONSerialization.jsonObject(with: outputData) {
                                outputJson = json
                            } else {
                                outputJson = ["result": execResult.output]
                            }
                        } else {
                            errorMessage = execResult.output
                        }

                        let resultMap: [String: Any] = [
                            "invocationId": invocationId ?? -1,
                            "status": execResult.status,
                            "output": outputJson as Any,
                            "outputRaw": execResult.output,
                            "errorMessage": errorMessage,
                            "logs": execResult.logs,
                            "durationMs": duration
                        ]

                        if let resultData = try? JSONSerialization.data(withJSONObject: resultMap),
                           let resultJson = String(data: resultData, encoding: .utf8) {
                            _ = try await connection.set(
                                RedisKey(RESULT_KEY_PREFIX + String(invocationId ?? -1)),
                                to: resultJson
                            ).get()
                            _ = try await connection.expire(
                                RedisKey(RESULT_KEY_PREFIX + String(invocationId ?? -1)),
                                after: .seconds(RESULT_TTL)
                            ).get()
                        }

                        print("Finished invocation: \(invocationId ?? -1) - \(execResult.status)")

                    } catch {
                        print("Error parsing job: \(error.localizedDescription)")

                        if let invocationId = invocationId {
                            let errorResult: [String: Any] = [
                                "invocationId": invocationId,
                                "status": "ERROR",
                                "output": NSNull(),
                                "outputRaw": "",
                                "errorMessage": error.localizedDescription,
                                "logs": "\(error)",
                                "durationMs": 0
                            ]

                            if let errorData = try? JSONSerialization.data(withJSONObject: errorResult),
                               let errorJson = String(data: errorData, encoding: .utf8) {
                                _ = try? await connection.set(
                                    RedisKey(RESULT_KEY_PREFIX + String(invocationId)),
                                    to: errorJson
                                ).get()
                                _ = try? await connection.expire(
                                    RedisKey(RESULT_KEY_PREFIX + String(invocationId)),
                                    after: .seconds(RESULT_TTL)
                                ).get()
                            }
                        }
                    }
                }
            } catch {
                print("Error in worker loop: \(error.localizedDescription)")
                try? await Task.sleep(nanoseconds: 1_000_000_000) // 1 second
            }
        }
    } catch {
        print("Fatal error: \(error.localizedDescription)")
    }
}

Task {
    await main()
}

RunLoop.main.run()
