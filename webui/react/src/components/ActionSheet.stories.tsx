import React, { useEffect, useState } from 'react';

import ActionSheet from './ActionSheet';

export default {
  component: ActionSheet,
  parameters: { layout: 'centered' },
  title: 'ActionSheet',
};

const ActionSheetContainer = () => {
  const [ isShowing, setIsShowing ] = useState(false);

  useEffect(() => {
    setIsShowing(true);
  }, []);

  return (
    <div style={{
      border: 'solid 1px #cccccc',
      height: 480,
      position: 'relative',
      width: 320,
    }}>
      <ActionSheet
        actions={[
          { icon: 'notebook', label: 'Launch Notebook' },
          { icon: 'notebook', label: 'Launch CPU-only Notebook' },
          { icon: 'logs', label: 'Master Logs', path: '/det/logs', popout: true },
          { icon: 'docs', label: 'Docs', path: '/docs', popout: true },
          { icon: 'cloud', label: 'API (Beta)', path: '/docs/rest-api/', popout: true },
        ]}
        show={isShowing} />
    </div>
  );
};

export const Default = (): React.ReactNode => <ActionSheetContainer />;
