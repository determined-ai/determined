import { execSync } from 'child_process';
import path from 'path';

if (process.env.PW_DET_PATH === undefined) {
  throw new Error('username must be defined');
}
if (process.env.PW_DET_MASTER === undefined) {
  throw new Error('password must be defined');
}

const detCommandBase = process.env.PW_DET_PATH;
const detCommandBaseOptions = ['-m', process.env.PW_DET_MASTER];
const detExecBase = `${detCommandBase} ${detCommandBaseOptions.join(' ')}`;
export function fullPath(relativePath: string): string {
  return path.join(process.cwd(), relativePath);
}

export function detAuthenticate(): string {
  detExecSync('user logout');
  try {
    return execSync(
      `echo ${process.env.PW_PASSWORD} | ${detExecBase} user login ${process.env.PW_USER_NAME}`,
      { stdio: 'pipe' },
    ).toString();
  } catch (e: unknown) {
    if (typeof e === 'object' && e !== null && 'stderr' in e && typeof e.stderr === 'string') {
      throw new Error('detExecSync error: ' + e.stderr);
    } else {
      throw e;
    }
  }
}

export function detExecSync(detCommand: string): string {
  try {
    return execSync(`${detExecBase} ${detCommand}`, { stdio: 'pipe' }).toString();
  } catch (e: unknown) {
    if (typeof e === 'object' && e !== null && 'stderr' in e && typeof e.stderr === 'string') {
      throw new Error('detExecSync error: ' + e.stderr);
    } else {
      throw e;
    }
  }
}
