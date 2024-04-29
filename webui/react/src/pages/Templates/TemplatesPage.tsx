import Button from 'hew/Button';
import { useModal } from 'hew/Modal';
import Row from 'hew/Row';
import React, { useRef } from 'react';

import Page from 'components/Page';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';

import TemplateCreateModalComponent from './TemplateCreateModal';

interface Props {
  workspaceId?: number;
}

const TemplatesPage: React.FC<Props> = ({ workspaceId }) => {
  const pageRef = useRef<HTMLElement>(null);
  const { canCreateTemplate } = usePermissions();
  const TemplateCreateModal = useModal(TemplateCreateModalComponent);
  return (
    <Page
      breadcrumb={
        workspaceId
          ? []
          : [
              {
                breadcrumbName: 'Manage. Templates',
                path: paths.templates(),
              },
            ]
      }
      containerRef={pageRef}
      id="templates"
      options={
        <Row>
          {canCreateTemplate && <Button onClick={TemplateCreateModal.open}>New Template</Button>}
        </Row>
      }
      title="Manage Templates">
      <TemplateCreateModal.Component workspaceId={workspaceId} />
    </Page>
  );
};

export default TemplatesPage;
