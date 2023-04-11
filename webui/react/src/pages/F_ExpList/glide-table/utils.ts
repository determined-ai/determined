import { DataEditorProps } from '@glideapps/glide-data-grid';
import dayjs from 'dayjs';

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

export const getTheme = (bodyStyles: CSSStyleDeclaration): DataEditorProps['theme'] => ({
  accentLight: '#FAFAFA',

  bgBubble: '#F5F5F5',
  // accentColor: '',
  // accentFg: '',
  // bgCell: '',
  // bgCellMedium: '',
  // bgIconHeader: '',
  bgHeader: '#FAFAFA',

  // fgIconHeader: '',
  // bgBubble: '',
  // textBubble: '',
  // bgBubbleSelected: '',
  // textDark: '',
  // bgHeaderHasFocus: '',
  // textLight: '',
  // bgHeaderHovered: '',
  // textMedium: '',
  // bgSearchResult: '',
  borderColor: '#00000000',

  fontFamily: bodyStyles.getPropertyValue('--theme-font-family'),

  // cellHorizontalPadding: 0,
  // textHeaderSelected: '',
  // cellVerticalPadding: 0,
  // drilldownBorder: '',
  // editorFontSize: '',
  headerBottomBorderColor: '#E0E0E0',

  headerFontStyle: 'normal 12px',

  linkColor: '#3875F6',

  textHeader: '#555555',
  // textGroupHeader: '',
  // baseFontStyle: '',

  // headerIconSize: 0,
  // horizontalBorderColor: '',
  // lineHeight: 0,
  // linkColor: '',
});

export const headerIcons: DataEditorProps['headerIcons'] = {
  selected:
    () => `<svg width="24" height="14" viewBox="0 0 24 14" fill="none" xmlns="http://www.w3.org/2000/svg">
  <rect x="0.5" y="0.5" width="13" height="13" rx="3.5" fill="#D9D9D9" fill-opacity="0.05" stroke="#454545"/>
  <line x1="3" y1="7" x2="11" y2="7" stroke="#929292" stroke-width="2"/>
  <path d="M20.9967 8.11333L18.9226 6L18 6.94L20.9967 10L24 6.94L23.0709 6L20.9967 8.11333Z" fill="#454545"/>
  </svg>
  `,
};
