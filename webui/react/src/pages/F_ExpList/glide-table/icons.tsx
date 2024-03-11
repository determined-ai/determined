import {
  DataEditorProps,
} from '@glideapps/glide-data-grid';
import { Theme } from 'hew/Theme';

// TODO: use theme here
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export const getHeaderIcons = (_appTheme: Theme): DataEditorProps['headerIcons'] => ({
  allSelected: () => `
    <svg width="16" height="16" viewBox="-1 -1 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
      <rect x="0.5" y="0.5" width="13" height="13" rx="3" fill="#D9D9D9" fill-opacity="0.05" stroke="#454545"/>
      <line x1="5.25" y1="6.5" x2="6.75" y2="8" stroke="#454545" stroke-width="1.5" stroke-linecap="round"/>
      <line x1="6.75" y1="8" x2="9.25" y2="5.5" stroke="#454545" stroke-width="1.5" stroke-linecap="round"/>
    </svg>
  `,
  noneSelected: () => `
    <svg width="16" height="16" viewBox="-1 -1 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
      <rect x="0.5" y="0.5" width="13" height="13" rx="3" fill="#D9D9D9" fill-opacity="0.05" stroke="#454545"/>
    </svg>
  `,
  someSelected: () => `
    <svg width="16" height="16" viewBox="-1 -1 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
      <rect x="0.5" y="0.5" width="13" height="13" rx="3" fill="#D9D9D9" fill-opacity="0.05" stroke="#454545"/>
      <line x1="3" y1="7" x2="11" y2="7" stroke="#929292" stroke-width="2"/>
    </svg>
  `,
});
