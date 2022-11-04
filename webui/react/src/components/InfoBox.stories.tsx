import React from 'react';

import InfoBox, { InfoRow } from './InfoBox';

export default {
  component: InfoBox,
  title: 'Determined/InfoBox',
};

const longText = `Lorem ipsum dolor sit amet, consectetur adipiscing elit. 
Duis at orci vel libero condimentum molestie. Cras in sem et diam faucibus 
ornare condimentum vitae nunc. Sed eu eros pulvinar, tristique nisi sit amet, 
pulvinar ante. Nulla non finibus justo.`;

const rows: InfoRow[] = [
  { content: 'Ipsum', label: 'Lorem' },
  { content: longText, label: 'Long Content' },
  { content: longText.split('.'), label: 'Array Content' },
  { content: 'Long Label', label: longText },
];

export const Default = (): React.ReactNode => <InfoBox rows={rows} />;
export const WithHeader = (): React.ReactNode => <InfoBox header="Header" rows={rows} />;
