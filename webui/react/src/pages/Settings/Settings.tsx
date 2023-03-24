import React from 'react';

import Page from 'components/Page';
import SettingsAccount from 'pages/Settings/SettingsAccount';

const Settings: React.FC = () => (
  <Page bodyNoPadding id="settings" stickyHeader title="Settings">
    <SettingsAccount />
  </Page>
);

export default Settings;
