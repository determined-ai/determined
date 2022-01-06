import React, { useEffect, useState } from 'react';

import { paths } from 'routes/utils';

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
        { icon: 'jupyter-lab', tag: 'Launch JupyterLab' },
        { icon: 'logs', tag: 'Cluster Logs', path: paths.clusterLogs(), popout: true },
        {
          external: true,
          icon: 'docs',
          tag: 'Docs',
          path: paths.docs(),
          popout: true,
        },
        {
          external: true,
          icon: 'cloud',
          tag: 'API (Beta)',
          path: paths.docs('/rest-api/'),
          popout: true,
        },
      ]}
      show={isShowing}
    />
  );
};

export const Default = (): React.ReactNode => <ActionSheetContainer />;
