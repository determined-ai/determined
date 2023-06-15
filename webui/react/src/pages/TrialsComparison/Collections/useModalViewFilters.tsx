import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import CodeEditor from 'components/kit/CodeEditor';
import useModal, { ModalHooks as Hooks } from 'hooks/useModal/useModal';
import { hasObjectKeys, isObject, isString } from 'utils/data';
import handleError from 'utils/error';
import { Loaded } from 'utils/loadable';

import { Ranker, TrialFilters, TrialSorter } from './filters';
import css from './useModalCreateCollection.module.scss';

export interface FilterModalProps {
  filters?: TrialFilters;
  initialModalProps?: ModalFuncProps;
  sorter?: TrialSorter;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (props: FilterModalProps) => void;
}

const useModalViewFilters = (): ModalHooks => {
  const [filters, setFilters] = useState<TrialFilters>();
  const [sorter, setSorter] = useState<TrialSorter>();

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal();

  const modalContent = useMemo(() => {
    const nonEmptyFilters = Object.entries(filters ?? {})
      .filter(([key, value]) =>
        key === 'ranker'
          ? (value as Ranker).rank && (value as Ranker).rank !== '0'
          : (key !== 'projectIds' && key !== 'workspaceIds' && Array.isArray(value)) ||
            isString(value)
          ? value.length > 0
          : isObject(value)
          ? hasObjectKeys(value)
          : false,
      )
      .map(([key, value]) => ({ [key]: value }))
      .sort()
      .reduce((a, b) => ({ ...a, ...b }), {});

    const sorterText = `Sort Order: ${sorter?.sortKey} ${
      sorter?.sortDesc ? 'descending' : 'ascending'
    }`;

    const hasFilters = !!Object.keys(nonEmptyFilters).length;

    const filtersText = hasFilters
      ? `Filters: ${JSON.stringify(nonEmptyFilters, null, 2)}`
      : 'Filters: empty';

    return (
      <div className={css.base}>
        <CodeEditor
          files={[{ content: Loaded([sorterText, filtersText].join('\n\n')), key: 'config.yaml' }]}
          height="100%"
          readonly
          onError={handleError}
        />
      </div>
    );
  }, [filters, sorter]);

  const getModalProps = useCallback((): ModalFuncProps => {
    const props = {
      closable: false,
      content: modalContent,
      icon: null,
      okCancel: false,
      title: 'Current View',
      width: 700,
    };
    return props;
  }, [modalContent]);

  const modalOpen = useCallback(
    ({ initialModalProps, filters, sorter }: FilterModalProps) => {
      setFilters(filters);
      setSorter(sorter);

      const newProps = {
        ...initialModalProps,
        ...getModalProps(),
      };
      openOrUpdate(newProps);
    },
    [getModalProps, openOrUpdate],
  );

  useEffect(() => {
    if (modalRef.current) {
      openOrUpdate(getModalProps());
    }
  }, [getModalProps, modalRef, openOrUpdate]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalViewFilters;
