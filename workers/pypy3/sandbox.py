import sys
import io
import json
import contextlib
import traceback

def run_code(code, input_data):
    # Capture stdout
    stdout_capture = io.StringIO()

    # Prepare local scope
    local_scope = {'event': input_data}

    status = "SUCCESS"
    output = ""
    logs = ""

    try:
        with contextlib.redirect_stdout(stdout_capture):
            # Execute user code
            exec(code, {}, local_scope)

            if 'handler' in local_scope and callable(local_scope['handler']):
                result = local_scope['handler'](input_data)
                # Try to serialize result as JSON
                try:
                    output = json.dumps(result, ensure_ascii=False)
                except (TypeError, ValueError):
                    output = str(result)
            else:
                status = "ERROR"
                output = "No 'handler(event)' function defined in code."

    except Exception as e:
        status = "ERROR"
        output = str(e)
        logs = traceback.format_exc()

    logs += stdout_capture.getvalue()

    return status, output, logs
