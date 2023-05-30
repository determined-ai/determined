import { DataEditorProps } from '@glideapps/glide-data-grid';
import dayjs from 'dayjs';

import { Theme } from 'shared/themes';
import {
  DURATION_MINUTE,
  DURATION_UNIT_MEASURES,
  DURATION_YEAR,
  durationInEnglish,
  getDuration,
} from 'shared/utils/datetime';
import { ExperimentItem } from 'types';

export const getDurationInEnglish = (record: ExperimentItem): string => {
  const duration = getDuration(record);
  const options = {
    conjunction: ' ',
    delimiter: ' ',
    largest: 2,
    serialComma: false,
    unitMeasures: { ...DURATION_UNIT_MEASURES, ms: 1000 },
  };
  return durationInEnglish(duration, options);
};

const JUST_NOW = 'Just Now';

const DATE_FORMAT = 'MMM D, YYYY';
export const getTimeInEnglish = (d: Date): string => {
  const options = {
    conjunction: ' ',
    delimiter: ' ',
    language: 'shortEn',
    largest: 1,
    serialComma: false,
    spacer: ' ',
  };

  const now = Date.now();
  const milliseconds = d.valueOf();
  const delta = milliseconds === undefined ? 0 : now - milliseconds;

  if (delta < DURATION_MINUTE) {
    return JUST_NOW;
  } else if (delta >= DURATION_YEAR) {
    return dayjs(milliseconds).format(DATE_FORMAT);
  } else {
    return `${durationInEnglish(delta, options)} ago`;
  }
};

/**
 * Glide Table Theme Reference
 * https://github.com/glideapps/glide-data-grid/blob/main/packages/core/API.md#theme
 */
export const getTheme = (appTheme: Theme): DataEditorProps['theme'] => {
  return {
    accentLight: appTheme.stageStrong,
    bgBubble: appTheme.ixStrong,
    bgCell: appTheme.stage,
    bgHeader: appTheme.surface,
    bgHeaderHovered: appTheme.surfaceStrong,
    borderColor: appTheme.ixBorder,
    fontFamily: appTheme.fontFamily,
    headerBottomBorderColor: appTheme.ixBorder,
    headerFontStyle: 'normal 12px',
    linkColor: appTheme.surfaceOn,
    textBubble: appTheme.surfaceBorderStrong,
    textDark: appTheme.surfaceOnWeak,
    textHeader: appTheme.surfaceOnWeak,
  };
};
