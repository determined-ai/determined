import { Button, Dropdown, Menu, Select } from 'antd';
import React, { MutableRefObject } from 'react';
import { useCallback, useEffect, useMemo, useState } from 'react';

import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import useSettings, { BaseType, SettingsConfig, SettingsHook } from 'hooks/useSettings';
import useStorage from 'hooks/useStorage';
import { deleteTrialsCollection, getTrialsCollections, patchTrialsCollection } from 'services/api';
import Icon from 'shared/components/Icon';
import { isNumber, numberElseUndefined } from 'shared/utils/data';
import { ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';

import { decodeTrialsCollection, encodeTrialsCollection } from '../api';

import { TrialsCollection } from './collections';
import EditButton from './EditButton';
import { FilterSetter, SetFilters, TrialFilters, TrialSorter } from './filters';
import useModalTrialCollection, { CollectionModalProps } from './useModalCreateCollection';
import css from './useTrialCollections.module.scss';

export interface TrialsCollectionInterface {
  collection: string;
  collections: TrialsCollection[];
  controls: JSX.Element;
  fetchCollections: () => Promise<TrialsCollection[] | undefined>;
  filters: TrialFilters;
  modalContextHolder: React.ReactElement;
  openCreateModal: (p: CollectionModalProps) => void;
  resetFilters: () => void;
  saveCollection: (name: string) => Promise<void>;
  setCollection: (name: string) => void;
  setFilters: SetFilters;
  setNewCollection: (c: TrialsCollection) => Promise<void>;
  sorter: TrialSorter;
}

const collectionStoragePath = (projectId: string) => `collection/${projectId}`;

const configForProject = (projectId: string): SettingsConfig => ({
  applicableRoutespace: '/trials',
  settings: [
    {
      defaultValue: '',
      key: 'collection',
      storageKey: 'collection',
      type: { baseType: BaseType.String },
    } ],
  storagePath: collectionStoragePath(projectId),
});

const getDefaultFilters = (projectId: string) => (
  { projectIds: [ String(projectId) ] }
);

const defaultSorter: TrialSorter = {
  sortDesc: true,
  sortKey: 'trialId',
};

type fx = () => void

export const useTrialCollections = (
  projectId: string,
  tableSettingsHook: SettingsHook<InteractiveTableSettings>,
  refetcher: MutableRefObject<fx | undefined>,
): TrialsCollectionInterface => {
  const { settings: tableSettings, updateSettings: updateTableSettings } = tableSettingsHook;
  const filterStorage = useStorage(`trial-filters}/${projectId ?? 1}`);
  const initFilters = filterStorage.getWithDefault<TrialFilters>(
    'filters',
    getDefaultFilters(projectId),
  );

  const [
    // eslint-disable-next-line array-element-newline
    filters,  // external filters
    _setFilters, // only use thru below wrapper
  ] = useState<TrialFilters>(initFilters);

  const setFilters = useCallback(
    (fs: FilterSetter) => {
      _setFilters((filters) => {
        if (!filters) return filters;
        const f = fs(filters);
        filterStorage.set('filters', f);
        return f;
      });
    },
    [ filterStorage ],
  );

  const sorter: TrialSorter = useMemo(() => ({
    ...defaultSorter,
    sortDesc: tableSettings.sortDesc,
    sortKey: tableSettings.sortKey ? String(tableSettings.sortKey) : '',
  }), [ tableSettings.sortDesc, tableSettings.sortKey ]);

  const resetFilters = useCallback(() => {
    filterStorage.remove('filters');
  }, [ filterStorage ]);

  const [ collections, setCollections ] = useState<TrialsCollection[]>([]);

  const settingsConfig = useMemo(() => configForProject(projectId), [ projectId ]);
  const { settings, updateSettings } = useSettings<{ collection: string }>(settingsConfig);

  const previousCollectionStorage = useStorage(`previous-collection/${projectId}`);

  const getPreviousCollection = useCallback(
    () => previousCollectionStorage.get('collection'),
    [ previousCollectionStorage ],
  );

  const setPreviousCollection = useCallback(
    (c) => previousCollectionStorage.set('collection', c),
    [ previousCollectionStorage ],
  );

  const setCollection = useCallback(
    (name: string) => {
      const _collection = collections.find((c) => c.name === name);
      if (_collection?.name != null) {
        updateSettings({ collection: _collection.name }, true);
      }
    },
    [ collections, updateSettings ],
  );

  const fetchCollections = useCallback(async () => {
    try {
      const id = parseInt(projectId);
      if (isNaN(id)) {
        const response = await getTrialsCollections(id);
        const collections = response.collections?.map(decodeTrialsCollection) ?? [];
        setCollections(collections);
        return collections;
      }
    } catch (e) {
      handleError(e, {
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to fetch collections.',
        silent: false,
        type: ErrorType.Api,
      });
    }
  }, [ projectId ]);

  useEffect(() => {
    fetchCollections();
  }, [ fetchCollections ]);

  const saveCollection = useCallback(async (name: string) => {
    try {
      const _collection = collections.find((c) => c.name === settings?.collection);
      const newCollection = { ..._collection, filters, name, sorter } as TrialsCollection;
      await patchTrialsCollection(encodeTrialsCollection(newCollection));
      fetchCollections();
      updateSettings({ collection: name }, true);
      setCollection(name);
    } catch (e) {
      handleError(e, {
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to save collection.',
        silent: false,
        type: ErrorType.Api,
      });
    }
  }, [
    collections,
    filters,
    sorter,
    fetchCollections,
    updateSettings,
    setCollection,
    settings?.collection,
  ]);

  const deleteCollection = useCallback(async () => {
    try {
      const _collection = collections.find((c) => c.name === settings?.collection);
      const id = numberElseUndefined(_collection?.id);
      if (isNumber(id)){
        await deleteTrialsCollection(id);
      }
      fetchCollections();
      setCollection(collections[0]?.name);
    } catch (e) {
      handleError(e, {
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to delete collection.',
        silent: false,
        type: ErrorType.Api,
      });
    }
  }, [ collections, fetchCollections, settings?.collection, setCollection ]);

  useEffect(() => {
    const _collection = collections.find((c) => c.name === settings?.collection);
    const previousCollection = getPreviousCollection();
    if (_collection && JSON.stringify(_collection) !== JSON.stringify(previousCollection)) {
      setFilters(() => _collection.filters);
      updateTableSettings({
        sortDesc: _collection.sorter.sortDesc,
        sortKey: _collection.sorter.sortKey,
      });
      setPreviousCollection(_collection);
    }
  }, [
    settings?.collection,
    collections,
    getPreviousCollection,
    setPreviousCollection,
    updateTableSettings,
    setFilters,
  ]);

  const setNewCollection = useCallback(
    async (newCollection?: TrialsCollection) => {
      if (!newCollection) return;
      try {
        const newCollections = await fetchCollections();
        const _collection = newCollections?.find((c) => c.name === newCollection.name);
        if (_collection?.name != null) {
          updateSettings({ collection: _collection.name }, true);
        }
        if (newCollection) setCollection(newCollection.name);
      } catch (e) {
        handleError(e, {
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to fetch new collection.',
          silent: false,
          type: ErrorType.Api,
        });
      }
      refetcher.current?.();
    },
    [ fetchCollections, setCollection, updateSettings, refetcher ],
  );

  const { modalOpen, contextHolder } = useModalTrialCollection({
    filters: filters,
    onConfirm: setNewCollection,
    projectId,
  });

  const createCollectionFromFilters = useCallback(() => {
    modalOpen({ trials: { filters, sorter } });
  }, [ filters, modalOpen, sorter ]);

  const resetFiltersToCollection = useCallback(() => {
    const filters = collections.find((c) => c.name === settings?.collection)?.filters;
    if (filters)
      setFilters(() => filters);

    const sorter = collections.find((c) => c.name === settings?.collection)?.sorter;
    if (sorter)
      updateTableSettings({ ...defaultSorter });

  }, [ settings?.collection, collections, updateTableSettings, setFilters ]);

  const clearFilters = useCallback(() => {
    const filters = collections.find((c) => c.name === settings?.collection)?.filters;
    if (filters)
      setFilters(() => ({ projectIds: [ projectId ] }));

  }, [ projectId, collections, settings?.collection, setFilters ]);

  const controls = (
    <div className={css.base}>
      <div className={css.options}>
        <Button onClick={createCollectionFromFilters}>New Collection</Button>
        <EditButton
          collectionName={settings?.collection}
          filters={filters}
          saveCollection={saveCollection}
        />
        <Select
          placeholder={collections?.length ? 'Select Collection' : 'No collections created'}
          value={settings.collection || undefined}
          onChange={(value) => setCollection(value)}>
          {[
            ...(collections?.map((collection) => (
              <Select.Option key={collection.name} value={collection.name}>
                {collection.name}
              </Select.Option>
            )) ?? []),
          ]}
        </Select>
        <Dropdown

          overlay={(
            <Menu
              items={[
                {
                  key: 'del',
                  label: 'Delete Collection',
                  onClick: deleteCollection,
                },
                {
                  key: 'res',
                  label: 'Restore Collection',
                  onClick: resetFiltersToCollection,
                },
                {
                  key: 'clr',
                  label: 'Clear Filters',
                  onClick: clearFilters,
                },
              ]}
            />
          )}
          trigger={[ 'click' ]}>
          <Button
            className={[ css.optionsDropdown, css.optionsDropdownThreeChild ].join(' ')}
            ghost
            icon={<Icon name="overflow-vertical" />}
          />
        </Dropdown>
      </div>
    </div>
  );

  return {
    collection: settings.collection,
    collections,
    controls,
    fetchCollections,
    filters,
    modalContextHolder: contextHolder,
    openCreateModal: modalOpen,
    resetFilters,
    saveCollection,
    setCollection,
    setFilters,
    setNewCollection,
    sorter,
  };
};
