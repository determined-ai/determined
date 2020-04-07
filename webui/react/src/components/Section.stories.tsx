import React from 'react';

import Section from './Section';

export default {
  component: Section,
  title: 'Section',
};

export const Default = (): React.ReactNode => <Section title="My Section" />;
export const WithOptions =
  (): React.ReactNode => <Section options={[ <button key="x" /> ]} title="My Section" />;
