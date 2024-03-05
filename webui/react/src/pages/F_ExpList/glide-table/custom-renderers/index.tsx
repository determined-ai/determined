import { DataEditorProps } from '@glideapps/glide-data-grid';

import CheckboxCell from './cells/checkboxCell';
import LinkCell from './cells/linkCell';
import LoadingCell from './cells/loadingCell';
import ProgressCell from './cells/progressCell';
import SparklineCell from './cells/sparklineCell';
import StateCell from './cells/stateCell';
import TagsCell from './cells/tagsCell';
import TextCell from './cells/textCell';
import UserProfileCell from './cells/userAvatarCell';

export const customRenderers: DataEditorProps['customRenderers'] = [
  CheckboxCell,
  StateCell,
  LinkCell,
  LoadingCell,
  ProgressCell,
  SparklineCell,
  TagsCell,
  TextCell,
  UserProfileCell,
];
