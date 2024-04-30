import { spawn } from 'child_process';

export const wait = (millis: number) => new Promise(resolve => setTimeout(resolve, millis));
export const makeCommand = async (command: string): Promise<boolean> => {
    console.log(`child make command ${command} starting`);
    const cmd = spawn('make', [command])
    return new Promise<boolean>((resolve) => {
        cmd.stdout.on('data', (data) => {
            console.log(`stdout: ${data}`);
        });
        cmd.stderr.on('data', (data) => {
            console.log(`stderr: ${data}`);
        });
        cmd.on('close', (code) => {
            console.log(`child make command ${command} exited with code ${code}`);
            resolve(code === 0);
        })
    });
}