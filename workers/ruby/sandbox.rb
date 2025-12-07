require 'json'
require 'timeout'
require 'stringio'

module Sandbox
  TIMEOUT_SECONDS = 30

  def self.run_code(code, input_data)
    logs = []
    status = "SUCCESS"
    output = ""

    # Create a new binding for isolated execution
    sandbox_binding = create_sandbox_binding(input_data, logs)

    begin
      Timeout.timeout(TIMEOUT_SECONDS) do
        # Execute user code in sandbox
        eval(code, sandbox_binding, "user_code.rb")

        # Check if handler method is defined and call it
        if sandbox_binding.local_variable_defined?(:handler_result)
          result = sandbox_binding.local_variable_get(:handler_result)
          output = result.to_json
        else
          # Try to call handler function if defined
          begin
            handler_proc = sandbox_binding.eval("method(:handler)")
            result = handler_proc.call(input_data)
            output = result.to_json
          rescue NameError
            status = "ERROR"
            output = "No 'handler' method defined in code. Please define: def handler(event) ... end"
          end
        end
      end
    rescue Timeout::Error
      status = "TIMEOUT"
      output = "Execution timed out after #{TIMEOUT_SECONDS} seconds"
    rescue SyntaxError => e
      status = "ERROR"
      output = "Syntax error: #{e.message}"
      logs << e.backtrace&.first(5)&.join("\n")
    rescue => e
      status = "ERROR"
      output = "Runtime error: #{e.message}"
      logs << e.backtrace&.first(10)&.join("\n")
    end

    [status, output, logs.flatten.compact.join("\n")]
  end

  private

  def self.create_sandbox_binding(input_data, logs)
    # Create isolated binding with limited access
    sandbox = Object.new

    # Define puts/print to capture logs
    sandbox.define_singleton_method(:puts) do |*args|
      logs << args.map(&:to_s).join(' ')
    end

    sandbox.define_singleton_method(:print) do |*args|
      logs << args.map(&:to_s).join('')
    end

    sandbox.define_singleton_method(:p) do |*args|
      logs << args.map(&:inspect).join(', ')
    end

    # Provide event data
    sandbox.define_singleton_method(:event) { input_data }

    sandbox.instance_eval { binding }
  end
end
