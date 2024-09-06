import Avatar from 'hew/Avatar';
import { Modal } from 'hew/Modal';
import { Label } from 'hew/Typography';
import yaml from 'js-yaml';
import React, { useMemo } from 'react';

import CodeEditor from 'components/CodeEditor';
import { NavigationItem } from 'components/NavigationSideBar';
import { paths } from 'routes/utils';
import { Template, Workspace } from 'types';
import handleError from 'utils/error';

interface Props {
  template?: Template;
  workspaces: Workspace[];
}

const TemplateViewModalComponent: React.FC<Props> = ({ template, workspaces }) => {
  const workspace = useMemo(() => {
    if (!template || !workspaces) return undefined;
    return workspaces.find((w) => w.id === template.workspaceId);
  }, [workspaces, template]);

  if (!template || !workspaces) return null;

  return (
    <Modal size="medium" title={`Template ${template.name}`}>
      <Label>Workspace</Label>
      {workspace && (
        <NavigationItem
          icon={<Avatar palette="muted" square text={workspace.name} />}
          label={workspace.name}
          path={paths.workspaceDetails(workspace.id)}
        />
      )}
      <Label>Config</Label>
      {template.config ? (
        <CodeEditor
          file={yaml.dump(template.config)}
          files={[{ key: 'template.yaml' }]}
          height="40vh"
          readonly
          onError={handleError}
        />
      ) : (
        'N/A'
      )}
    </Modal>
  );
};

export default TemplateViewModalComponent;
