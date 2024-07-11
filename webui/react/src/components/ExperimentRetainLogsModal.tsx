import Checkbox, { CheckboxChangeEvent } from 'hew/Checkbox';
import Form from 'hew/Form';
import InputNumber from 'hew/InputNumber';
import { Modal } from 'hew/Modal';
import { useToast } from 'hew/Toast';
import React, { useCallback, useId, useState } from 'react';

import useFeature from 'hooks/useFeature';
import { changeExperimentLogRetention } from 'services/api';
import { V1BulkExperimentFilters } from 'services/api-ts-sdk';
import handleError from 'utils/error';
import { pluralizer } from 'utils/string';

const FORM_ID = 'retain-experiment-logs-form';
const FOREVER = -1;
const MAX_DAYS = 32767;

type FormInputs = {
  numDays: number;
};

interface Props {
  excludedExperimentIds?: Map<number, unknown>;
  experimentIds: number[];
  projectId: number;
  filters?: V1BulkExperimentFilters;
  onSubmit?: (successfulIds?: number[]) => void;
}

const ExperimentRetainLogsModalComponent: React.FC<Props> = ({
  excludedExperimentIds,
  experimentIds,
  projectId,
  filters,
  onSubmit,
}: Props) => {
  const idPrefix = useId();
  const { openToast } = useToast();
  const [checked, setChecked] = useState<boolean>(false);
  const [form] = Form.useForm<FormInputs>();
  const inputDays = Form.useWatch('numDays', form);
  const f_flat_runs = useFeature().isOn('flat_runs');

  const pluralize = useCallback(
    (n: number) =>
      pluralizer.apply(null, [
        n,
        ...(f_flat_runs ? (['search', 'searches'] as const) : (['experiment'] as const)),
      ]),
    [f_flat_runs],
  );

  const handleCheckBoxChange = useCallback(
    (event: CheckboxChangeEvent) => {
      const isChecked = event.target.checked;
      setChecked(isChecked);
      if (isChecked) {
        form.setFieldValue('numDays', FOREVER);
      } else {
        form.setFieldValue('numDays', null);
      }
    },
    [form],
  );

  const handleSubmit = useCallback(async () => {
    const values = await form.validateFields();
    const numberDays = values.numDays;
    let filt = filters;
    if (excludedExperimentIds && excludedExperimentIds.size > 0) {
      filt = { ...filters, excludedExperimentIds: Array.from(excludedExperimentIds.keys()) };
    }
    try {
      const results = await changeExperimentLogRetention({
        experimentIds,
        filters: filt,
        numDays: numberDays,
        projectId,
      });

      onSubmit?.(results.successful);

      const numSuccesses = results.successful.length;
      const numFailures = results.failed.length;

      const stringDays = numberDays === -1 ? 'forever' : `for ${numberDays} days`;

      if (numFailures === 0) {
        openToast({
          closeable: true,
          description: `Retained logs for ${results.successful.length} ${pluralize(numSuccesses)} ${stringDays}`,
          title: 'Retain Logs Success',
        });
      } else if (numSuccesses === 0) {
        openToast({
          description: `Unable to retain logs for ${numFailures} ${pluralize(numFailures)}`,
          severity: 'Error',
          title: 'Retain Logs Failure',
        });
      } else {
        openToast({
          closeable: true,
          description: `Failed to retain logs for ${numFailures} ${pluralize(2)} out of ${
            numFailures + numSuccesses
          } for ${numberDays} days.`,
          severity: 'Warning',
          title: 'Partial Retain Logs Failure',
        });
      }
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to retain logs' });
    }
  }, [
    form,
    filters,
    excludedExperimentIds,
    experimentIds,
    projectId,
    onSubmit,
    openToast,
    pluralize,
  ]);

  const actionCopy =
    filters !== undefined ? 'Retain Logs' : `Retain logs for ${pluralize(experimentIds.length)}`;

  return (
    <Modal
      cancel
      size="small"
      submit={{
        disabled: inputDays == null || inputDays < FOREVER || inputDays > MAX_DAYS,
        form: idPrefix + FORM_ID,
        handleError,
        handler: handleSubmit,
        text: actionCopy,
      }}
      title={actionCopy}>
      <Form form={form} id={idPrefix + FORM_ID} layout="vertical">
        <Form.Item
          label="Number of Days"
          name="numDays"
          rules={[
            {
              max: MAX_DAYS,
              message: 'Number of days is required',
              min: FOREVER,
              required: true,
              type: 'number',
            },
          ]}>
          <InputNumber disabled={checked} precision={0} />
        </Form.Item>
        <Checkbox checked={checked} onChange={handleCheckBoxChange}>
          Forever
        </Checkbox>
      </Form>
    </Modal>
  );
};

export default ExperimentRetainLogsModalComponent;
