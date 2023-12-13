import { type, TypeOf } from 'io-ts';

import { Mode } from 'components/ThemeProvider';
import { valueof } from 'utils/valueof';

export const settings = type({
  mode: valueof(Mode),
});
export type Settings = TypeOf<typeof settings>;
export const STORAGE_PATH = 'settings-theme';
