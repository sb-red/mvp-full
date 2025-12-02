const redis = require('redis');
const sandbox = require('./sandbox');

const REDIS_HOST = process.env.REDIS_HOST || 'localhost';
const REDIS_PORT = parseInt(process.env.REDIS_PORT || '6379', 10);

const QUEUE_KEY = 'execution_queue:javascript';
const RESULT_KEY_PREFIX = 'result:';
const RESULT_TTL = 600; // 10 minutes

async function main() {
    const client = redis.createClient({
        socket: {
            host: REDIS_HOST,
            port: REDIS_PORT,
        },
    });

    client.on('error', (err) => console.error('Redis Client Error:', err));

    await client.connect();
    console.log(`JS Worker started. Connected to Redis at ${REDIS_HOST}:${REDIS_PORT}`);

    while (true) {
        try {
            // Blocking pop from queue (5 second timeout)
            const item = await client.brPop(QUEUE_KEY, 5);

            if (item) {
                const rawData = item.element;
                let invocationId = null;

                try {
                    const data = JSON.parse(rawData);
                    invocationId = data.invocationId; // Now int64
                    const code = data.code;
                    const inputData = data.input || {};

                    console.log(`Processing invocation: ${invocationId}`);

                    const startTime = Date.now();
                    const { status, output, logs } = sandbox.runCode(code, inputData);
                    const duration = Date.now() - startTime;

                    // Parse output as JSON if possible
                    let outputJson = null;
                    let errorMessage = '';
                    if (status === 'SUCCESS') {
                        try {
                            outputJson = typeof output === 'string' ? JSON.parse(output) : output;
                        } catch {
                            outputJson = { result: output };
                        }
                    } else {
                        errorMessage = output;
                    }

                    const result = {
                        invocationId,
                        status,
                        output: outputJson,
                        outputRaw: typeof output === 'string' ? output : JSON.stringify(output),
                        errorMessage,
                        logs,
                        durationMs: duration,
                    };

                    await client.set(
                        RESULT_KEY_PREFIX + invocationId,
                        JSON.stringify(result),
                        { EX: RESULT_TTL }
                    );

                    console.log(`Finished invocation: ${invocationId} - ${status}`);
                } catch (parseError) {
                    console.error('Error parsing job:', parseError.message);

                    if (invocationId) {
                        const errorResult = {
                            invocationId,
                            status: 'ERROR',
                            output: null,
                            outputRaw: '',
                            errorMessage: parseError.message,
                            logs: parseError.stack || '',
                            durationMs: 0,
                        };
                        await client.set(
                            RESULT_KEY_PREFIX + invocationId,
                            JSON.stringify(errorResult),
                            { EX: RESULT_TTL }
                        );
                    }
                }
            }
        } catch (err) {
            console.error('Error in worker loop:', err.message);
            // Small delay before retrying on error
            await new Promise((resolve) => setTimeout(resolve, 1000));
        }
    }
}

main().catch(console.error);
