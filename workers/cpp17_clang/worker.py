import os
import redis
import json
import time
import sandbox

REDIS_HOST = os.environ.get('REDIS_HOST', 'localhost')
REDIS_PORT = int(os.environ.get('REDIS_PORT', 6379))

r = redis.Redis(host=REDIS_HOST, port=REDIS_PORT, db=0)

QUEUE_KEY = "execution_queue:cpp17_clang"
RESULT_KEY_PREFIX = "result:"

def main():
    print(f"C++17 (Clang) Worker started. Connecting to Redis at {REDIS_HOST}:{REDIS_PORT}")

    while True:
        item = r.brpop(QUEUE_KEY, timeout=5)

        if item:
            _, raw_data = item
            invocation_id = None
            try:
                data = json.loads(raw_data)
                invocation_id = data['invocationId']
                code = data['code']
                input_data = data.get('input', {})

                print(f"Processing invocation: {invocation_id}")

                start_time = time.time()
                status, output, logs = sandbox.run_code(code, input_data)
                duration = int((time.time() - start_time) * 1000)

                output_json = None
                error_message = ""
                if status == "SUCCESS":
                    try:
                        output_json = json.loads(output) if output else None
                    except json.JSONDecodeError:
                        output_json = {"result": output}
                else:
                    error_message = output

                result = {
                    'invocationId': invocation_id,
                    'status': status,
                    'output': output_json,
                    'outputRaw': output,
                    'errorMessage': error_message,
                    'logs': logs,
                    'durationMs': duration
                }

                r.set(RESULT_KEY_PREFIX + str(invocation_id), json.dumps(result), ex=600)
                print(f"Finished invocation: {invocation_id} - {status}")

            except json.JSONDecodeError as e:
                print(f"Error: Invalid JSON in queue: {e}")

            except KeyError as e:
                print(f"Error: Missing required field in request: {e}")
                if invocation_id:
                    error_result = {
                        'invocationId': invocation_id,
                        'status': 'ERROR',
                        'output': None,
                        'outputRaw': '',
                        'errorMessage': f'Invalid request format: missing {e}',
                        'logs': '',
                        'durationMs': 0
                    }
                    r.set(RESULT_KEY_PREFIX + str(invocation_id), json.dumps(error_result), ex=600)

            except Exception as e:
                print(f"Error processing job: {e}")
                if invocation_id:
                    import traceback
                    error_result = {
                        'invocationId': invocation_id,
                        'status': 'ERROR',
                        'output': None,
                        'outputRaw': '',
                        'errorMessage': str(e),
                        'logs': traceback.format_exc(),
                        'durationMs': 0
                    }
                    r.set(RESULT_KEY_PREFIX + str(invocation_id), json.dumps(error_result), ex=600)

if __name__ == "__main__":
    main()
