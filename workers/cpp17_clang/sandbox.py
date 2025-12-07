import subprocess
import os
import json
import shutil
import uuid

COMPILE_TIMEOUT = 30
EXECUTION_TIMEOUT = 30

WRAPPER_TEMPLATE = '''#include <iostream>
#include <fstream>
#include <sstream>
#include <nlohmann/json.hpp>

using json = nlohmann::json;

// Forward declaration
json handler(const json& event);

{USER_CODE}

int main() {{
    try {{
        std::stringstream buffer;
        buffer << std::cin.rdbuf();
        json event = json::parse(buffer.str());

        json result = handler(event);

        std::cout << result.dump() << std::endl;
        return 0;
    }} catch (const std::exception& e) {{
        std::cerr << "Error: " << e.what() << std::endl;
        return 1;
    }}
}}
'''

def compile_code(code: str, work_dir: str) -> tuple:
    source_file = os.path.join(work_dir, "main.cpp")
    binary_file = os.path.join(work_dir, "main")

    full_code = WRAPPER_TEMPLATE.replace("{USER_CODE}", code)

    with open(source_file, 'w') as f:
        f.write(full_code)

    try:
        result = subprocess.run(
            ["clang++", "-std=c++17", "-O2", "-o", binary_file, source_file,
             "-I/usr/include", "-I/usr/local/include"],
            capture_output=True,
            text=True,
            timeout=COMPILE_TIMEOUT
        )

        if result.returncode != 0:
            return False, "", f"Compilation error:\n{result.stderr}"

        return True, binary_file, result.stderr

    except subprocess.TimeoutExpired:
        return False, "", "Compilation timed out"
    except Exception as e:
        return False, "", str(e)


def run_code(code: str, input_data: dict) -> tuple:
    status = "SUCCESS"
    output = ""
    logs = ""

    work_dir = os.path.join("/tmp/sandbox", str(uuid.uuid4()))
    os.makedirs(work_dir, exist_ok=True)

    try:
        success, binary_path, compile_logs = compile_code(code, work_dir)
        logs += compile_logs

        if not success:
            return "ERROR", compile_logs, "Compilation failed"

        logs += "Compilation successful\n"

        input_json = json.dumps(input_data)

        result = subprocess.run(
            [binary_path],
            input=input_json,
            capture_output=True,
            text=True,
            timeout=EXECUTION_TIMEOUT,
            cwd=work_dir
        )

        if result.returncode != 0:
            status = "ERROR"
            output = result.stderr or "Execution failed with non-zero exit code"
        else:
            output = result.stdout.strip()

        if result.stderr:
            logs += result.stderr

    except subprocess.TimeoutExpired:
        status = "TIMEOUT"
        output = f"Execution timed out after {EXECUTION_TIMEOUT} seconds"
    except Exception as e:
        status = "ERROR"
        output = str(e)
    finally:
        shutil.rmtree(work_dir, ignore_errors=True)

    return status, output, logs
