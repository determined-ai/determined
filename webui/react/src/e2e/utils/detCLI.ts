import { execSync } from 'child_process';
import path from 'path';

import { detMasterURL, detPath, password, username } from './envVars';

export function fullPath(relativePath: string): string {
  return path.join(process.cwd(), relativePath);
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
