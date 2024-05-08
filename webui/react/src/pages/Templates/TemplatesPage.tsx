import React, { useRef } from 'react';

import Page from 'components/Page';
import { paths } from 'routes/utils';

import TemplateList from './TemplatesList';

const TemplatesPage: React.FC = () => {
  const pageRef = useRef<HTMLElement>(null);

  return (
    <Page
      breadcrumb={[
        {
          breadcrumbName: 'Manage Templates',
          path: paths.templates(),
        },
      ]}
      containerRef={pageRef}
      id="templates"
      title="Manage Templates">
      <TemplateList />
    </Page>
  );
};

export default TemplatesPage;
