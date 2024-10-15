import Spinner from 'hew/Spinner';
import React, { useRef } from 'react';

import ConfigPolicies from 'components/ConfigPolicies';
import Page from 'components/Page';
import PageNotFound from 'components/PageNotFound';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import determinedStore, { BrandingType } from 'stores/determinedInfo';
import { useObservable } from 'utils/observable';

const TemplatesPage: React.FC = () => {
  const pageRef = useRef<HTMLElement>(null);
  const info = useObservable(determinedStore.info);
  const {
    canViewGlobalConfigPolicies,
    loading: rbacLoading,
  } = usePermissions();

  const canView = info.branding === BrandingType.HPE && canViewGlobalConfigPolicies;

  if (rbacLoading) return <Spinner spinning />;

  if (!canView) return <PageNotFound />;

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
