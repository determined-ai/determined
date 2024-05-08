import { execSync } from 'child_process';
import path from 'path';

if (process.env.PW_DET_PATH === undefined) {
  throw new Error('username must be defined');
}
if (process.env.PW_DET_MASTER === undefined) {
  throw new Error('password must be defined');
}

export function fullPath(relativePath: string): string {
  return path.join(process.cwd(), relativePath);
}

export function detExecSync(detCommand: string): string {
  try {
    return execSync(`${process.env.PW_DET_PATH} ${detCommand}`, {
      env: {
        ...process.env,
        DET_MASTER: process.env.PW_DET_MASTER,
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
