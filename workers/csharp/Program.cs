using System;
using System.Text.Json;
using StackExchange.Redis;
using Newtonsoft.Json;
using Newtonsoft.Json.Linq;

namespace CSharpWorker;

class Program
{
    private const string QueueKey = "execution_queue:csharp";
    private const string ResultKeyPrefix = "result:";
    private static readonly TimeSpan ResultTTL = TimeSpan.FromMinutes(10);

    static void Main(string[] args)
    {
        var redisHost = Environment.GetEnvironmentVariable("REDIS_HOST") ?? "localhost";
        var redisPort = Environment.GetEnvironmentVariable("REDIS_PORT") ?? "6379";

        Console.WriteLine($"C# Worker started. Connecting to Redis at {redisHost}:{redisPort}");

        var redis = ConnectionMultiplexer.Connect($"{redisHost}:{redisPort}");
        var db = redis.GetDatabase();

        Console.WriteLine("Connected to Redis successfully");

        while (true)
        {
            try
            {
                // Block and wait for job from queue (timeout 5 seconds)
                var result = db.ListRightPop(QueueKey);

                if (result.IsNullOrEmpty)
                {
                    Thread.Sleep(1000);
                    continue;
                }

                var rawData = result.ToString();
                JObject? requestObj = null;
                long invocationId = 0;

                try
                {
                    requestObj = JObject.Parse(rawData);
                    invocationId = requestObj["invocationId"]?.Value<long>() ?? 0;
                    var code = requestObj["code"]?.ToString() ?? "";
                    var input = requestObj["input"] ?? new JObject();

                    Console.WriteLine($"Processing invocation: {invocationId}");

                    var startTime = DateTime.UtcNow;
                    var (status, output, logs) = Executor.RunCode(code, input);
                    var durationMs = (long)(DateTime.UtcNow - startTime).TotalMilliseconds;

                    object? outputParsed = null;
                    string errorMessage = "";

                    if (status == "SUCCESS")
                    {
                        if (!string.IsNullOrEmpty(output))
                        {
                            try
                            {
                                outputParsed = JsonConvert.DeserializeObject(output);
                            }
                            catch
                            {
                                outputParsed = new { result = output };
                            }
                        }
                    }
                    else
                    {
                        errorMessage = output;
                    }

                    var execResult = new
                    {
                        invocationId = invocationId,
                        status = status,
                        output = outputParsed,
                        outputRaw = output,
                        errorMessage = errorMessage,
                        logs = logs,
                        durationMs = durationMs
                    };

                    var resultJson = JsonConvert.SerializeObject(execResult);
                    var resultKey = $"{ResultKeyPrefix}{invocationId}";
                    db.StringSet(resultKey, resultJson, ResultTTL);

                    Console.WriteLine($"Finished invocation: {invocationId} - {status}");
                }
                catch (JsonReaderException ex)
                {
                    Console.WriteLine($"Error parsing request JSON: {ex.Message}");
                    continue;
                }
                catch (Exception ex)
                {
                    Console.WriteLine($"Error processing job: {ex.Message}");
                    if (invocationId > 0)
                    {
                        var errorResult = new
                        {
                            invocationId = invocationId,
                            status = "ERROR",
                            output = (object?)null,
                            outputRaw = "",
                            errorMessage = ex.Message,
                            logs = ex.StackTrace ?? "",
                            durationMs = 0L
                        };
                        var resultKey = $"{ResultKeyPrefix}{invocationId}";
                        db.StringSet(resultKey, JsonConvert.SerializeObject(errorResult), ResultTTL);
                    }
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Redis error: {ex.Message}");
                Thread.Sleep(5000);
            }
        }
    }
}
