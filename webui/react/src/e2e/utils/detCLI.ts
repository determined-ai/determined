import { execSync } from 'child_process';
import path from 'path';

import { detMasterURL } from './envVars';

export function fullPath(relativePath: string): string {
  return path.join(process.cwd(), relativePath);
}

export function detExecSync(detCommand: string): string {
  try {
    return execSync(`${process.env.PW_DET_PATH || 'det'} ${detCommand}`, {
      env: {
        ...process.env,
        DET_MASTER: detMasterURL(),
        DET_PASS: process.env.PW_PASSWORD,
        DET_USER: process.env.PW_USER_NAME,
      },
      stdio: 'pipe',
      timeout: 5_000,
    }).toString();
  } catch (e: unknown) {
    if (typeof e === 'object' && e !== null && 'stderr' in e && typeof e.stderr === 'string') {
      throw new Error('detExecSync error: ' + e.stderr);
    } else {
      throw e;
    }
  }
}
