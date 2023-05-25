import React from 'react';

import Page from 'components/Page';
import SettingsAccount from 'pages/Settings/SettingsAccount';
import { paths } from 'routes/utils';

const Settings: React.FC = () => (
  <Page
    bodyNoPadding
    breadcrumb={[
      {
        breadcrumbName: 'Settings',
        path: paths.settings(),
      },
    ]}
    id="settings"
    stickyHeader
    title="Settings">
    <SettingsAccount />
  </Page>
);

export default Settings;
