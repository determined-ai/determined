import { DataEditorProps } from '@glideapps/glide-data-grid';

import CheckboxCell from './cells/checkboxCell';
import ExperimentStateCell from './cells/experimentStateCell';
import LinkCell from './cells/linkCell';
import ProgressCell from './cells/progressCell';
import SparklineCell from './cells/sparklineCell';
import SpinnerCell from './cells/spinnerCell';
import TagsCell from './cells/tagsCell';
import UserProfileCell from './cells/userAvatarCell';

export const customRenderers: DataEditorProps['customRenderers'] = [
  SparklineCell,
  TagsCell,
  UserProfileCell,
  SpinnerCell,
  ProgressCell,
  CheckboxCell,
  LinkCell,
  ExperimentStateCell,
];
