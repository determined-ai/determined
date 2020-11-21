import { Button, Space } from 'antd';
import React from 'react';

import Section from './Section';

export default {
  component: Section,
  title: 'Section',
};

export const Default = (): React.ReactNode => (
  <Section title="Default Section">Section Content</Section>
);

export const WithOptions = (): React.ReactNode => (
  <Section
    options={<Space>
      <Button key="1">Option 1</Button>
      <Button key="2">Option 2</Button>
    </Space>}
    title="Section with Content">
      Section Content
  </Section>
);
