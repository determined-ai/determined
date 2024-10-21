import { exec as execCallback, execSync } from 'child_process';
import path from 'path';
import { promisify } from 'util';

import { detMasterURL, detPath, password, username } from './envVars';

const exec = promisify(execCallback);

const baseDir = execSync('git rev-parse --show-toplevel').toString().trim();

export function fullPath(relativePath: string): string {
  return path.join(baseDir, relativePath);
}

export function detExecSync(detCommand: string): string {
  try {
    return execSync(`${detPath()} ${detCommand}`, {
      env: {
        ...process.env,
        DET_MASTER: detMasterURL(),
        DET_PASS: password(),
        DET_USER: username(),
      },
      stdio: 'pipe',
      timeout: 10_000,
    }).toString();
  } catch (e: unknown) {
    if (typeof e === 'object' && e !== null && 'stderr' in e && typeof e.stderr === 'string') {
      throw new Error('detExecSync error: ' + e.stderr);
    } else {
      throw e;
    }
  }
}

export const detExec = async (command: string): Promise<string> => {
  try {
    const child = await exec(`${detPath()} ${command}`, {
      env: {
        ...process.env,
        DET_MASTER: detMasterURL(),
        DET_PASS: password(),
        DET_USER: username(),
      },
      timeout: 10_000,
    });
    return child.stdout;
  } catch (e: unknown) {
    if (e && typeof e === 'object' && 'stderr' in e && typeof e.stderr === 'string') {
      throw new Error(`detExec error: ${e.stderr}`);
    } else {
      throw e;
    }
  }
};
