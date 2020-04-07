import { hasKey } from 'utils/types';
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isPropTrue = (props: Record<string, any>, propName: string): boolean => {
  if (hasKey(props, propName)) {
    return !!props[propName];
  }
  return false;
};
