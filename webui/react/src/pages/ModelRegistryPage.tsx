import React, { useRef } from 'react';

import ModelRegistry from 'components/ModelRegistry';
import Page from 'components/Page';
import { paths } from 'routes/utils';

const ModelRegistryPage: React.FC = () => {
  const pageRef = useRef<HTMLElement>(null);

  return (
    <Page
      breadcrumb={[
        {
          breadcrumbName: 'Model Registry',
          path: paths.modelList(),
        },
      ]}
      containerRef={pageRef}
      id="models"
      title="Model Registry">
      <ModelRegistry />
    </Page>
  );
};

export default ModelRegistryPage;
