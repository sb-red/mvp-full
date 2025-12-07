require 'redis'
require 'json'
require_relative 'sandbox'

REDIS_HOST = ENV['REDIS_HOST'] || 'localhost'
REDIS_PORT = (ENV['REDIS_PORT'] || 6379).to_i

redis = Redis.new(host: REDIS_HOST, port: REDIS_PORT)

QUEUE_KEY = "execution_queue:ruby"
RESULT_KEY_PREFIX = "result:"

puts "Ruby Worker started. Connecting to Redis at #{REDIS_HOST}:#{REDIS_PORT}"

loop do
  begin
    result = redis.brpop(QUEUE_KEY, timeout: 5)
    next unless result

    _, raw_data = result
    invocation_id = nil

    begin
      data = JSON.parse(raw_data)
      invocation_id = data['invocationId']
      code = data['code']
      input_data = data['input'] || {}

      puts "Processing invocation: #{invocation_id}"

      start_time = Time.now
      status, output, logs = Sandbox.run_code(code, input_data)
      duration = ((Time.now - start_time) * 1000).to_i

      output_json = nil
      error_message = ""

      if status == "SUCCESS"
        begin
          output_json = JSON.parse(output) if output && !output.empty?
        rescue JSON::ParserError
          output_json = { "result" => output }
        end
      else
        error_message = output
      end

      result_data = {
        invocationId: invocation_id,
        status: status,
        output: output_json,
        outputRaw: output.to_s,
        errorMessage: error_message,
        logs: logs,
        durationMs: duration
      }

      redis.set("#{RESULT_KEY_PREFIX}#{invocation_id}", result_data.to_json, ex: 600)
      puts "Finished invocation: #{invocation_id} - #{status}"

    rescue JSON::ParserError => e
      puts "Error: Invalid JSON in queue: #{e.message}"

    rescue KeyError => e
      puts "Error: Missing required field: #{e.message}"
      if invocation_id
        error_result = {
          invocationId: invocation_id,
          status: 'ERROR',
          output: nil,
          outputRaw: '',
          errorMessage: "Invalid request format: #{e.message}",
          logs: '',
          durationMs: 0
        }
        redis.set("#{RESULT_KEY_PREFIX}#{invocation_id}", error_result.to_json, ex: 600)
      end

    rescue => e
      puts "Error processing job: #{e.message}"
      if invocation_id
        error_result = {
          invocationId: invocation_id,
          status: 'ERROR',
          output: nil,
          outputRaw: '',
          errorMessage: e.message,
          logs: e.backtrace&.join("\n") || '',
          durationMs: 0
        }
        redis.set("#{RESULT_KEY_PREFIX}#{invocation_id}", error_result.to_json, ex: 600)
        puts "Saved error result for invocation: #{invocation_id}"
      end
    end

  rescue Redis::ConnectionError => e
    puts "Redis connection error: #{e.message}. Retrying in 1 second..."
    sleep 1
  end
end
