import React, { useRef } from 'react';

import ConfigPolicies from 'components/ConfigPolicies';
import Page from 'components/Page';
import { paths } from 'routes/utils';

const TemplatesPage: React.FC = () => {
  const pageRef = useRef<HTMLElement>(null);

  return (
    <Page
      breadcrumb={[
        {
          breadcrumbName: 'Config Policies',
          path: paths.configPolicies(),
        },
      ]}
      containerRef={pageRef}
      id="configPolicies"
      title="Config Policies">
      <ConfigPolicies global />
    </Page>
  );
};

export default TemplatesPage;
