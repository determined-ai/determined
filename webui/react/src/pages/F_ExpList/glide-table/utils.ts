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
    language: 'en',
    largest: 1,
    serialComma: false,
    spacer: ' ',
  };

  const now = Date.now();

  const milliseconds = d.valueOf();

  const delta = milliseconds === undefined ? 0 : now - milliseconds;

  return delta < DURATION_MINUTE
    ? JUST_NOW
    : delta >= DURATION_YEAR
    ? dayjs(milliseconds).format(DATE_FORMAT)
    : `${durationInEnglish(delta, options)} ago`;
};

export const getTheme = (appTheme: Theme): DataEditorProps['theme'] => {
  return {
    accentLight: appTheme.stage,
    bgBubble: appTheme.ixStrong,
    bgCell: appTheme.stageWeak,
    bgHeader: appTheme.surface,
    bgHeaderHovered: appTheme.surfaceStrong,
    borderColor: '#00000000',
    fontFamily: appTheme.fontFamily,
    headerBottomBorderColor: appTheme.stageStrong,
    headerFontStyle: 'normal 12px',
    // horizontalBorderColor: appTheme.stageStrong,
    linkColor: appTheme.statusActive,
    textBubble: appTheme.stageBorderStrong,
    textDark: appTheme.stageOnWeak,
    textHeader: appTheme.stageOnWeak,
    // bgBubble: '#F5F5F5',
    // accentColor: '',
    // accentFg: '',
    // bgHeaderHasFocus: '',
    // textLight: '',
    // bgCellMedium: '',
    // bgIconHeader: '',
    // fgIconHeader: '',
    // bgBubbleSelected: '',
    // textMedium: '',
    // bgSearchResult: '',
    // cellHorizontalPadding: 0,
    // textHeaderSelected: '',
    // cellVerticalPadding: 0,
    // drilldownBorder: '',
    // editorFontSize: '',
    // textGroupHeader: '',
    // baseFontStyle: '',
    // headerIconSize: 0,
    // horizontalBorderColor: '',
    // lineHeight: 0,
    // linkColor: '',
  };
};
