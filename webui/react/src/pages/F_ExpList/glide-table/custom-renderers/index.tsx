import { DataEditorProps } from '@hpe.com/glide-data-grid';

import CheckboxCell from './cells/checkboxCell';
import ExperimentStateCell from './cells/experimentStateCell';
import LinkCell from './cells/linkCell';
import LoadingCell from './cells/loadingCell';
import ProgressCell from './cells/progressCell';
import SparklineCell from './cells/sparklineCell';
import TagsCell from './cells/tagsCell';
import UserProfileCell from './cells/userAvatarCell';

export const customRenderers: DataEditorProps['customRenderers'] = [
  CheckboxCell,
  ExperimentStateCell,
  LinkCell,
  LoadingCell,
  ProgressCell,
  SparklineCell,
  TagsCell,
  UserProfileCell,
];
