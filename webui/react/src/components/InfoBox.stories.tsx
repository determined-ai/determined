import React from 'react';

import InfoBox, { InfoRow } from './InfoBox';

export default {
  component: InfoBox,
  title: 'InfoBox',
};

const longText = `Lorem ipsum dolor sit amet, consectetur adipiscing elit. 
Duis at orci vel libero condimentum molestie. Cras in sem et diam faucibus 
ornare condimentum vitae nunc. Sed eu eros pulvinar, tristique nisi sit amet, 
pulvinar ante. Nulla non finibus justo.`;

const rows: InfoRow[] = [
  { content: 'Ipsum', tag: 'Lorem' },
  { content: longText, tag: 'Long Content' },
  { content: longText.split('.'), tag: 'Array Content' },
  { content: 'Long Label', tag: longText },
];

export const Default = (): React.ReactNode => <InfoBox rows={rows} />;
export const WithHeader = (): React.ReactNode => <InfoBox header="Header" rows={rows} />;
