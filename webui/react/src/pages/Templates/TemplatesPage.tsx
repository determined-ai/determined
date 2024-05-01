import Button from 'hew/Button';
import Message from 'hew/Message';
import { useModal } from 'hew/Modal';
import Row from 'hew/Row';
import { ErrorType } from 'hew/utils/error';
import _ from 'lodash';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import Page from 'components/Page';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { getTaskTemplates } from 'services/api';
import { Template } from 'types';
import handleError from 'utils/error';

import TemplateCreateModalComponent from './TemplateCreateModal';
import TemplateList from './TemplatesList';

interface Props {
  workspaceId?: number;
}

const TemplatesPage: React.FC<Props> = ({ workspaceId }) => {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [canceler] = useState(new AbortController());

  const pageRef = useRef<HTMLElement>(null);
  const { canCreateTemplate } = usePermissions();
  const TemplateCreateModal = useModal(TemplateCreateModalComponent);

  const fetchTemplates = useCallback(async () => {
    try {
      const templates = await getTaskTemplates({}, { signal: canceler.signal });
      setTemplates((prev) => {
        if (_.isEqual(prev, templates)) return prev;
        return templates;
      });
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to fetch webhooks.',
        silent: true,
        type: ErrorType.Api,
      });
    } finally {
      setIsLoading(false);
    }
  }, [canceler.signal]);

  useEffect(() => {
    setIsLoading(true);
    fetchTemplates();
  }, [fetchTemplates]);

  useEffect(() => {
    return () => canceler.abort();
  }, [canceler]);

  return (
    <Page
      breadcrumb={
        workspaceId
          ? []
          : [
              {
                breadcrumbName: 'Manage Templates',
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
      {templates.length === 0 && !isLoading ? (
        <Message
          description="Move settings that are shared by many tasks into a single YAML file, that can then be referenced by configurations that require those settings."
          icon="columns"
          title="No Template Configured"
        />
      ) : (
        <TemplateList isLoading={isLoading} pageRef={pageRef} templates={templates} />
      )}
      <TemplateCreateModal.Component workspaceId={workspaceId} onSuccess={fetchTemplates} />
    </Page>
  );
};

export default TemplatesPage;
