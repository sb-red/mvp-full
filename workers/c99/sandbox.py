import subprocess
import os
import json
import shutil
import uuid

COMPILE_TIMEOUT = 30
EXECUTION_TIMEOUT = 30

WRAPPER_TEMPLATE = '''#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <cjson/cJSON.h>

// Forward declaration
cJSON* handler(cJSON* event);

{USER_CODE}

int main() {{
    char buffer[65536];
    size_t len = 0;
    int c;

    while ((c = getchar()) != EOF && len < sizeof(buffer) - 1) {{
        buffer[len++] = c;
    }}
    buffer[len] = '\\0';

    cJSON* event = cJSON_Parse(buffer);
    if (event == NULL) {{
        fprintf(stderr, "Error: Failed to parse input JSON\\n");
        return 1;
    }}

    cJSON* result = handler(event);
    if (result == NULL) {{
        fprintf(stderr, "Error: handler returned NULL\\n");
        cJSON_Delete(event);
        return 1;
    }}

    char* output = cJSON_Print(result);
    printf("%s\\n", output);

    free(output);
    cJSON_Delete(result);
    cJSON_Delete(event);

    return 0;
}}
'''

def compile_code(code: str, work_dir: str) -> tuple:
    source_file = os.path.join(work_dir, "main.c")
    binary_file = os.path.join(work_dir, "main")

    full_code = WRAPPER_TEMPLATE.replace("{USER_CODE}", code)

    with open(source_file, 'w') as f:
        f.write(full_code)

    try:
        result = subprocess.run(
            ["gcc", "-std=c99", "-O2", "-o", binary_file, source_file,
             "-I/usr/include", "-I/usr/local/include",
             "-L/usr/lib", "-L/usr/local/lib", "-lcjson"],
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
