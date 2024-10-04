import Alert from 'hew/Alert';
import Button from 'hew/Button';
import CodeEditor from 'hew/CodeEditor';
import Column from 'hew/Column';
import Form, { hasErrors } from 'hew/Form';
import Pivot, { PivotProps } from 'hew/Pivot';
import Row from 'hew/Row';
import Spinner from 'hew/Spinner';
import { useToast } from 'hew/Toast';
import useConfirm from 'hew/useConfirm';
import { Loadable, NotLoaded } from 'hew/utils/loadable';
import yaml from 'js-yaml';
import { isEmpty } from 'lodash';
import { useState } from 'react';

import { useAsync } from 'hooks/useAsync';
import usePermissions from 'hooks/usePermissions';
import {
  deleteWorkspaceConfigPolicies,
  getWorkspaceConfigPolicies,
  updateWorkspaceConfigPolicies,
} from 'services/api';
import handleError from 'utils/error';

interface Props {
  workspaceId?: number;
}

type ConfigPoliciesType = 'experiments' | 'tasks';

interface ConfigPoliciesValues {
  label: string;
  workloadType: 'NTSC' | 'EXPERIMENT';
}

const ConfigPoliciesValues: Record<ConfigPoliciesType, ConfigPoliciesValues> = {
  experiments: {
    label: 'Experiments',
    workloadType: 'EXPERIMENT',
  },
  tasks: {
    label: 'Tasks',
    workloadType: 'NTSC',
  },
};

interface TabProps {
  workspaceId?: number;
  type: ConfigPoliciesType;
}

type FormInputs = {
  configPolicies: string;
};

const ConfigPolicies: React.FC<Props> = ({ workspaceId }: Props) => {
  const tabItems: PivotProps['items'] = [
    {
      children: <ConfigPoliciesTab type="experiments" workspaceId={workspaceId} />,
      key: 'experiments',
      label: ConfigPoliciesValues.experiments.label,
    },
    {
      children: <ConfigPoliciesTab type="tasks" workspaceId={workspaceId} />,
      key: 'tasks',
      label: ConfigPoliciesValues.tasks.label,
    },
  ];

  return <Pivot items={tabItems} type="secondary" />;
};

const ConfigPoliciesTab: React.FC<TabProps> = ({ workspaceId, type }: TabProps) => {
  const confirm = useConfirm();
  const { openToast } = useToast();
  const { canModifyWorkspaceConfigPolicies, loading: rbacLoading } = usePermissions();
  const [form] = Form.useForm<FormInputs>();

  const [disabled, setDisabled] = useState(true);

  const APPLY_MESSAGE = "You're about to apply these config policies to this workspace.";
  const VIEW_MESSAGE = 'An admin applied these config policies to this workspace.';
  const CONFIRMATION_MESSAGE = 'Config policies updated';

  const updatePolicies = async () => {
    if (workspaceId) {
      const configPolicies = form.getFieldValue('configPolicies');
      if (configPolicies.length) {
        try {
          await updateWorkspaceConfigPolicies({
            configPolicies,
            workloadType: ConfigPoliciesValues[type].workloadType,
            workspaceId,
          });
          openToast({ title: CONFIRMATION_MESSAGE });
        } catch (error) {
          handleError(error);
        }
      } else {
        try {
          await deleteWorkspaceConfigPolicies({
            workloadType: ConfigPoliciesValues[type].workloadType,
            workspaceId,
          });
          openToast({ title: CONFIRMATION_MESSAGE });
        } catch (error) {
          handleError(error);
        }
      }
    }
  };

  const confirmApply = () => {
    confirm({
      content: (
        <span>
          This will impact{' '}
          <strong>
            <u>all</u>
          </strong>{' '}
          underlying projects and their experiments in this workspace.
        </span>
      ),
      okText: 'Apply',
      onConfirm: updatePolicies,
      onError: handleError,
      size: 'medium',
      title: APPLY_MESSAGE,
    });
  };

  const loadableConfigPolicies: Loadable<string | undefined> = useAsync(async () => {
    if (workspaceId) {
      const response = await getWorkspaceConfigPolicies({
        workloadType: ConfigPoliciesValues[type].workloadType,
        workspaceId,
      });
      if (isEmpty(response.configPolicies)) return undefined;
      return response.configPolicies;
    }
    return NotLoaded;
  }, [workspaceId, type]);

  const initialConfigPoliciesYAML = yaml.dump(loadableConfigPolicies.getOrElse(undefined));

  const handleChange = () => {
    setDisabled(
      hasErrors(form) || form.getFieldValue('configPolicies') === initialConfigPoliciesYAML,
    );
  };

  if (rbacLoading) return <Spinner spinning />;

  return (
    <Column>
      <Row width="fill">
        <div style={{ width: '100%' }}>
          {canModifyWorkspaceConfigPolicies ? (
            <Alert
              action={
                <Button disabled={disabled} onClick={confirmApply}>
                  Apply
                </Button>
              }
              message={APPLY_MESSAGE}
              showIcon
            />
          ) : (
            <Alert message={VIEW_MESSAGE} showIcon />
          )}
        </div>
      </Row>
      <Row width="fill">
        <div style={{ width: '100%' }}>
          <Form form={form} onFieldsChange={handleChange}>
            <Form.Item
              name="configPolicies"
              rules={[
                {
                  validator: (_, value) => {
                    try {
                      yaml.load(value);
                      return Promise.resolve();
                    } catch (err: unknown) {
                      return Promise.reject(
                        new Error(
                          `Invalid YAML on line ${(err as { mark: { line: string } }).mark.line}.`,
                        ),
                      );
                    }
                  },
                },
              ]}>
              <CodeEditor
                file={initialConfigPoliciesYAML}
                files={[{ key: type, title: `${type}-config-policies.yaml` }]}
                readonly={!canModifyWorkspaceConfigPolicies}
                onError={(error) => {
                  handleError(error);
                }}
              />
            </Form.Item>
          </Form>
        </div>
      </Row>
    </Column>
  );
};

export default ConfigPolicies;
