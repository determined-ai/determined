import React, { useRef } from 'react';

import Page from 'components/Page';
import { paths } from 'routes/utils';

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
      title="Manage Templates"
    />
  );
};

export default TemplatesPage;
