import React, { useEffect } from 'react';

import JupyterLabSettings from 'components/JupyterLab.settings';
import JupyterLabModalComponent from 'components/JupyterLabModal';
import { useModal } from 'components/kit/Modal';
import { keyEmitter, KeyEvent, ShortcutConfig, shortcutMatch } from 'hooks/useKeyTracker';
import { useSettings } from 'hooks/useSettings';
import { JupyterLabOptions } from 'utils/jupyter';

interface Props {
  active?: boolean;
}

const JupyterLabGlobal: React.FC<Props> = ({ active }) => {
  const JupyterLabModal = useModal(JupyterLabModalComponent);
  const { settings } = useSettings<JupyterLabOptions>(JupyterLabSettings);

  useEffect(() => {
    const shortcut: ShortcutConfig | undefined = settings.shortcut
      ? JSON.parse(settings.shortcut)
      : undefined;

    const keyDownListener = (e: KeyboardEvent) => {
      if (shortcut && shortcutMatch(e, shortcut)) {
        JupyterLabModal.open();
      }
    };

    if (active) keyEmitter.on(KeyEvent.KeyDown, keyDownListener);

    return () => {
      keyEmitter.off(KeyEvent.KeyDown, keyDownListener);
    };
  }, [JupyterLabModal, settings, active]);

  return <JupyterLabModal.Component />;
};

export default JupyterLabGlobal;
