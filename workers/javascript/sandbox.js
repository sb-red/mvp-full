const vm = require('vm');

function runCode(code, inputData) {
    const logs = [];
    let status = 'SUCCESS';
    let output = '';

    // Create a custom console that captures logs
    const customConsole = {
        log: (...args) => logs.push(args.map(String).join(' ')),
        error: (...args) => logs.push('[ERROR] ' + args.map(String).join(' ')),
        warn: (...args) => logs.push('[WARN] ' + args.map(String).join(' ')),
        info: (...args) => logs.push(args.map(String).join(' ')),
    };

    // Sandbox context with limited globals
    const context = {
        event: inputData,
        console: customConsole,
        JSON: JSON,
        Math: Math,
        Date: Date,
        Array: Array,
        Object: Object,
        String: String,
        Number: Number,
        Boolean: Boolean,
        parseInt: parseInt,
        parseFloat: parseFloat,
        isNaN: isNaN,
        isFinite: isFinite,
    };

    try {
        vm.createContext(context);

        // Execute user code to define handler function
        vm.runInContext(code, context, {
            timeout: 30000, // 30 second timeout
            displayErrors: true,
        });

        // Check if handler function exists
        if (typeof context.handler !== 'function') {
            status = 'ERROR';
            output = "No 'handler(event)' function defined in code.";
        } else {
            // Call handler function
            const result = context.handler(inputData);
            output = result !== undefined ? JSON.stringify(result) : 'undefined';
        }
    } catch (err) {
        status = 'ERROR';
        output = err.message;
        logs.push(err.stack || err.toString());
    }

    return {
        status,
        output,
        logs: logs.join('\n'),
    };
}

module.exports = { runCode };
