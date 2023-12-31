import Button from 'hew/Button';
import { shortcutToString } from 'hew/InputShortcut';
import { useModal } from 'hew/Modal';
import Row from 'hew/Row';
import Tooltip from 'hew/Tooltip';
import React from 'react';

import JupyterLabModalComponent from 'components/JupyterLabModal';
import NewJupyterLabModalComponent from 'components/JupyterLabModal2';
import shortCutSettingsConfig, {
  Settings as ShortcutSettings,
} from 'components/UserSettings.settings';
import { useSettings } from 'hooks/useSettings';
import { Workspace } from 'types';
interface Props {
  enabled?: boolean;
  workspace?: Workspace;
}

const JupyterLabButton: React.FC<Props> = ({ enabled, workspace }: Props) => {
  const JupyterLabModal = useModal(JupyterLabModalComponent);
  const NewJupyterLabModal = useModal(NewJupyterLabModalComponent);
  const {
    settings: { jupyterLab: jupyterLabShortcut },
  } = useSettings<ShortcutSettings>(shortCutSettingsConfig);

  return (
    <>
      {enabled ? (
        <Row>
          <Tooltip content={shortcutToString(jupyterLabShortcut)}>
            <Button onClick={JupyterLabModal.open}>Launch JupyterLab</Button>
          </Tooltip>
          <Button onClick={NewJupyterLabModal.open}>Launch JupyterLab (new form)</Button>
        </Row>
      ) : (
        <Tooltip content="You do not have permission to launch JupyterLab" placement="leftBottom">
          <div>
            <Button disabled>Launch JupyterLab</Button>
          </div>
        </Tooltip>
      )}
      <JupyterLabModal.Component workspace={workspace} />
      <NewJupyterLabModal.Component workspace={workspace} />
    </>
  );
};

export default JupyterLabButton;
