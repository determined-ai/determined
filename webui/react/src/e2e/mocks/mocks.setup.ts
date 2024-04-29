import { spawn } from 'child_process';

import { test } from '@playwright/test';

test.describe('Start Mocks', async () => {
    makeCommand('mb-start');
    await new Promise(resolve => setTimeout(resolve, 3000));
    makeCommand('mb-record-imposters');
});


function makeCommand(command: string) {
    const cmd = spawn('make', [command])
    cmd.stdout.on('data', (data) => {
        console.log(`stdout: ${data}`);
    });
    cmd.stderr.on('data', (data) => {
        console.log(`stdout: ${data}`);
    });
}
