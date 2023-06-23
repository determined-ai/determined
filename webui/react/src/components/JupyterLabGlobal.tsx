import React, { useEffect } from 'react';

import JupyterLabModalComponent from 'components/JupyterLabModal';
import { useModal } from 'components/kit/Modal';
import { keyEmitter, KeyEvent } from 'hooks/useKeyTracker';
import { useSettings } from 'hooks/useSettings';
import shortCutSettingsConfig, {
  Settings as ShortcutSettings,
} from 'pages/Settings/UserSettings.settings';
import { matchesShortcut } from 'utils/shortcut';
interface Props {
  active?: boolean;
}

const JupyterLabGlobal: React.FC<Props> = ({ active }) => {
  const JupyterLabModal = useModal(JupyterLabModalComponent);
  const {
    settings: { jupyterLab: jupyterLabShortcut },
  } = useSettings<ShortcutSettings>(shortCutSettingsConfig);

  useEffect(() => {
    const keyDownListener = (e: KeyboardEvent) => {
      if (matchesShortcut(e, jupyterLabShortcut)) {
        JupyterLabModal.open();
      }
    };

    if (active) keyEmitter.on(KeyEvent.KeyDown, keyDownListener);

    return () => {
      keyEmitter.off(KeyEvent.KeyDown, keyDownListener);
    };
  }, [JupyterLabModal, jupyterLabShortcut, active]);

  return <JupyterLabModal.Component />;
};

export default JupyterLabGlobal;
