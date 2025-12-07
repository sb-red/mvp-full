using System;
using System.Diagnostics;
using System.IO;
using System.Text;
using Newtonsoft.Json;
using Newtonsoft.Json.Linq;

namespace CSharpWorker;

public static class Executor
{
    private const int CompileTimeoutMs = 60000;
    private const int ExecutionTimeoutMs = 30000;

    private static string GetWrapperCode(string userCode)
    {
        return $@"using System;
using System.Text.Json;
using System.IO;

{userCode}

class Program {{
    static void Main(string[] args) {{
        try {{
            string input = Console.In.ReadToEnd();
            JsonElement evt = JsonDocument.Parse(input).RootElement;

            object result = Handler.Run(evt);

            string output = JsonSerializer.Serialize(result);
            Console.WriteLine(output);
        }} catch (Exception e) {{
            Console.Error.WriteLine($""Error: {{e.Message}}"");
            Environment.Exit(1);
        }}
    }}
}}
";
    }

    private const string CsprojTemplate = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net7.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
  </PropertyGroup>
</Project>
";

    public static (string status, string output, string logs) RunCode(string code, JToken inputData)
    {
        var workDir = Path.Combine("/tmp/sandbox", Guid.NewGuid().ToString());
        var logs = new StringBuilder();

        try
        {
            Directory.CreateDirectory(workDir);

            // Write .csproj
            var csprojPath = Path.Combine(workDir, "Handler.csproj");
            File.WriteAllText(csprojPath, CsprojTemplate);

            // Write Program.cs with wrapper
            var fullCode = GetWrapperCode(code);
            var programPath = Path.Combine(workDir, "Program.cs");
            File.WriteAllText(programPath, fullCode);

            // Compile
            var compileResult = RunProcess("dotnet", "build -c Release -o bin", workDir, CompileTimeoutMs);
            logs.AppendLine(compileResult.stderr);

            if (compileResult.exitCode != 0)
            {
                return ("ERROR", $"Compilation error:\n{compileResult.stdout}\n{compileResult.stderr}", logs.ToString());
            }

            logs.AppendLine("Compilation successful");

            // Run the compiled DLL
            var dllPath = Path.Combine(workDir, "bin", "Handler.dll");
            var inputJson = inputData.ToString();

            var runResult = RunProcessWithInput("dotnet", dllPath, inputJson, workDir, ExecutionTimeoutMs);

            if (!string.IsNullOrEmpty(runResult.stderr))
            {
                logs.AppendLine(runResult.stderr);
            }

            if (runResult.timedOut)
            {
                return ("TIMEOUT", $"Execution timed out after {ExecutionTimeoutMs / 1000} seconds", logs.ToString());
            }

            if (runResult.exitCode != 0)
            {
                var errorMsg = string.IsNullOrEmpty(runResult.stderr)
                    ? "Execution failed with non-zero exit code"
                    : runResult.stderr;
                return ("ERROR", errorMsg, logs.ToString());
            }

            return ("SUCCESS", runResult.stdout.Trim(), logs.ToString());
        }
        catch (Exception ex)
        {
            return ("ERROR", ex.Message, logs.ToString());
        }
        finally
        {
            try
            {
                if (Directory.Exists(workDir))
                {
                    Directory.Delete(workDir, true);
                }
            }
            catch { }
        }
    }

    private static (int exitCode, string stdout, string stderr) RunProcess(string command, string args, string workDir, int timeoutMs)
    {
        using var process = new Process
        {
            StartInfo = new ProcessStartInfo
            {
                FileName = command,
                Arguments = args,
                WorkingDirectory = workDir,
                RedirectStandardOutput = true,
                RedirectStandardError = true,
                UseShellExecute = false,
                CreateNoWindow = true
            }
        };

        process.Start();

        var stdout = process.StandardOutput.ReadToEnd();
        var stderr = process.StandardError.ReadToEnd();

        if (!process.WaitForExit(timeoutMs))
        {
            process.Kill();
            return (-1, stdout, "Process timed out");
        }

        return (process.ExitCode, stdout, stderr);
    }

    private static (int exitCode, string stdout, string stderr, bool timedOut) RunProcessWithInput(
        string command, string args, string input, string workDir, int timeoutMs)
    {
        using var process = new Process
        {
            StartInfo = new ProcessStartInfo
            {
                FileName = command,
                Arguments = args,
                WorkingDirectory = workDir,
                RedirectStandardInput = true,
                RedirectStandardOutput = true,
                RedirectStandardError = true,
                UseShellExecute = false,
                CreateNoWindow = true
            }
        };

        process.Start();
        process.StandardInput.Write(input);
        process.StandardInput.Close();

        var stdout = process.StandardOutput.ReadToEnd();
        var stderr = process.StandardError.ReadToEnd();

        if (!process.WaitForExit(timeoutMs))
        {
            process.Kill();
            return (-1, stdout, "Execution timed out", true);
        }

        return (process.ExitCode, stdout, stderr, false);
    }
}
