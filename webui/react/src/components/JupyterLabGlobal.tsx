import React, { useEffect } from 'react';

import JupyterLabSettings from 'components/JupyterLab.settings';
import JupyterLabModalComponent from 'components/JupyterLabModal';
import { useModal } from 'components/kit/Modal';
import { keyEmitter, KeyEvent, ShortcutConfig, shortcutMatch } from 'hooks/useKeyTracker';
import { useSettings } from 'hooks/useSettings';
import { JupyterLabOptions } from 'utils/jupyter';

const JupyterLabGlobal: React.FC = () => {
  const JupyterLabModal = useModal(JupyterLabModalComponent);
  const { settings } = useSettings<JupyterLabOptions>(JupyterLabSettings);

  useEffect(() => {
    const shortcut: ShortcutConfig = settings.shortcut ? JSON.parse(settings.shortcut) : undefined;

    const keyDownListener = (e: KeyboardEvent) => {
      if (shortcutMatch(e, shortcut)) {
        JupyterLabModal.open();
      }
    };

    keyEmitter.on(KeyEvent.KeyDown, keyDownListener);

    return () => {
      keyEmitter.off(KeyEvent.KeyDown, keyDownListener);
    };
  }, [JupyterLabModal, settings]);

  return <JupyterLabModal.Component />;
};

export default JupyterLabGlobal;
