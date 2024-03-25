import { DataEditorProps } from '@glideapps/glide-data-grid';

import CheckboxCellRenderer, {
  CHECKBOX_CELL,
  CheckboxCell,
  getCheckboxDimensions,
} from './cells/checkboxCell';
import LinkCellRenderer, { LINK_CELL, LinkCell } from './cells/linkCell';
import LoadingCellRenderer, { LOADING_CELL, LoadingCell } from './cells/loadingCell';
import ProgressCellRenderer, { PROGRESS_CELL, ProgressCell } from './cells/progressCell';
import SparklineCellRenderer, { SPARKLINE_CELL, SparklineCell } from './cells/sparklineCell';
import StateCellRenderer, { State, STATE_CELL, StateCell } from './cells/stateCell';
import TagsCellRenderer, { TAGS_CELL, TagsCell } from './cells/tagsCell';
import TextCellRenderer, { TEXT_CELL, TextCell } from './cells/textCell';
import UserAvatarCellRenderer, { USER_AVATAR_CELL, UserAvatarCell } from './cells/userAvatarCell';
import { drawArrow, drawTextWithEllipsis, roundedRect } from './utils';

const customRenderers: DataEditorProps['customRenderers'] = [
  CheckboxCellRenderer,
  StateCellRenderer,
  LinkCellRenderer,
  LoadingCellRenderer,
  ProgressCellRenderer,
  SparklineCellRenderer,
  TagsCellRenderer,
  TextCellRenderer,
  UserAvatarCellRenderer,
];

export {
  customRenderers,
  CHECKBOX_CELL,
  type CheckboxCell,
  getCheckboxDimensions,
  LINK_CELL,
  type LinkCell,
  LOADING_CELL,
  type LoadingCell,
  PROGRESS_CELL,
  type ProgressCell,
  SPARKLINE_CELL,
  type SparklineCell,
  STATE_CELL,
  type StateCell,
  State,
  TAGS_CELL,
  type TagsCell,
  TEXT_CELL,
  type TextCell,
  USER_AVATAR_CELL,
  type UserAvatarCell,
  drawArrow,
  drawTextWithEllipsis,
  roundedRect,
};
