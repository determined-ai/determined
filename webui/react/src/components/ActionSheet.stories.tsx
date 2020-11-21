import React, { useEffect, useState } from 'react';

import ActionSheet from './ActionSheet';

export default {
  component: ActionSheet,
  parameters: { layout: 'fullscreen' },
  title: 'ActionSheet',
};

const ActionSheetContainer = () => {
  const [ isShowing, setIsShowing ] = useState(false);

  useEffect(() => {
    setIsShowing(true);
  }, []);

  return (
    <ActionSheet
      actions={[
        { icon: 'notebook', label: 'Launch Notebook' },
        { icon: 'notebook', label: 'Launch CPU-only Notebook' },
        { icon: 'logs', label: 'Master Logs', path: '/logs', popout: true },
        {
          external: true,
          icon: 'docs',
          label: 'Docs',
          path: '/docs',
          popout: true,
        },
        {
          external: true,
          icon: 'cloud',
          label: 'API (Beta)',
          path: '/docs/rest-api/',
          popout: true,
        },
      ]}
      show={isShowing} />
  );
};

export const Default = (): React.ReactNode => <ActionSheetContainer />;
