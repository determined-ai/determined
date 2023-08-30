import { DataEditorProps } from '@hpe.com/glide-data-grid';

import CheckboxCell from 'pages/F_ExpList/glide-table/custom-renderers/cells/checkboxCell';
import ExperimentStateCell from 'pages/F_ExpList/glide-table/custom-renderers/cells/experimentStateCell';
import LinkCell from 'pages/F_ExpList/glide-table/custom-renderers/cells/linkCell';
import LoadingCell from 'pages/F_ExpList/glide-table/custom-renderers/cells/loadingCell';
import ProgressCell from 'pages/F_ExpList/glide-table/custom-renderers/cells/progressCell';
import SparklineCell from 'pages/F_ExpList/glide-table/custom-renderers/cells/sparklineCell';
import TagsCell from 'pages/F_ExpList/glide-table/custom-renderers/cells/tagsCell';
import UserProfileCell from 'pages/F_ExpList/glide-table/custom-renderers/cells/userAvatarCell';

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
