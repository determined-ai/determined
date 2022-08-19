import { Button, Select } from 'antd';
import React, { ReactNode } from 'react';
import { Dispatch, SetStateAction, useCallback, useEffect, useMemo, useState } from 'react';

import useSettings, { BaseType, SettingsConfig } from 'hooks/useSettings';
import useStorage from 'hooks/useStorage';
import { getTrialsCollections, patchTrialsCollection } from 'services/api';
import {
  V1OrderBy,
} from 'services/api-ts-sdk';
import { isNumber } from 'shared/utils/data';

import { decodeTrialsCollection, encodeTrialsCollection } from '../api';

import { TrialsCollection } from './collections';
import { FilterSetter, SetFilters, TrialFilters, TrialSorter } from './filters';
import useModalTrialCollection, { CollectionModalProps } from './useModalCreateCollection';

export interface TrialsCollectionInterface {
  collection: string;
  collections: TrialsCollection[];
  controls: ReactNode;
  fetchCollections: () => Promise<TrialsCollection[] | undefined>;
  filters: TrialFilters;
  modalContextHolder: React.ReactElement;
  openCreateModal: (p: CollectionModalProps) => void;
  resetFilters: () => void;
  saveCollection: () => Promise<void>;
  setCollection: (name: string) => void;
  setFilters: SetFilters;
  setNewCollection: (c: TrialsCollection) => Promise<void>;
  setSorter: Dispatch<SetStateAction<TrialSorter>>
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

export const useTrialCollections = (projectId: string): TrialsCollectionInterface => {
  const filterStorage = useStorage(`trial-filters}/${projectId ?? 1}`);
  const initFilters = filterStorage.getWithDefault<TrialFilters>(
    'filters',
    getDefaultFilters(projectId),
  );

  const [ sorter, setSorter ] = useState<TrialSorter>({
    orderBy: V1OrderBy.ASC,
    sortKey: 'trialId',
  });

  const [ filters, _setFilters ] = useState<TrialFilters>(initFilters);

  const setFilters = useCallback((fs: FilterSetter) => {
    _setFilters((filters) => {
      if (!filters) return filters;
      const f = fs(filters);
      filterStorage.set('filters', f);
      return f;
    });
  }, [ filterStorage ]);

  const resetFilters = useCallback(() => {
    filterStorage.remove('filters');
  }, [ filterStorage ]);

  const [ collections, setCollections ] = useState<TrialsCollection[]>([]);

  const settingsConfig = useMemo(() => configForProject(projectId), [ projectId ]);
  const { settings, updateSettings } =
  useSettings<{collection: string}>(settingsConfig);

  const previousCollectionStorage = useStorage(`previous-collection/${projectId}`);

  const getPreviousCollection = useCallback(
    () => previousCollectionStorage.get('collection'),
    [ previousCollectionStorage ],
  );

  const setPreviousCollection = useCallback(
    (c) => previousCollectionStorage.set('collection', c),
    [ previousCollectionStorage ],
  );

  const setCollection = useCallback((name: string) => {
    const _collection = collections.find((c) => c.name === name);
    if (_collection?.name != null) {
      updateSettings({ collection: _collection.name });
    }
  }, [ collections, updateSettings ]);

  const fetchCollections = useCallback(async () => {
    const id = parseInt(projectId);
    if (isNumber(id)) {
      const response = await getTrialsCollections(id);
      const collections = response.collections?.map(decodeTrialsCollection) ?? [];
      setCollections(collections);
      return collections;
    }
  }, [ projectId ]);

  useEffect(() => {
    fetchCollections();
  }, [ fetchCollections ]);

  const saveCollection = useCallback(async () => {
    const _collection = collections.find((c) => c.name === settings?.collection);
    const newCollection = { ..._collection, filters, sorter } as TrialsCollection;
    await patchTrialsCollection(encodeTrialsCollection(newCollection));
    fetchCollections();

  }, [ collections, filters, settings?.collection, sorter, fetchCollections ]);

  useEffect(() => {
    const _collection = collections.find((c) => c.name === settings?.collection);
    const previousCollection = getPreviousCollection();
    if (_collection && (JSON.stringify(_collection) !== JSON.stringify(previousCollection))) {
      _setFilters(_collection.filters);
      setPreviousCollection(_collection);
    }
  }, [ settings?.collection, collections, getPreviousCollection, setPreviousCollection ]);

  const setNewCollection = useCallback(async (newCollection?: TrialsCollection) => {
    if (!newCollection) return;
    try {
      const newCollections = await fetchCollections();
      const _collection = newCollections?.find((c) => c.name === newCollection.name);
      if (_collection?.name != null) {
        updateSettings({ collection: _collection.name });
      }
      if (newCollection) setCollection(newCollection.name);
    } catch {
      // duly noted
    }
  }, [ fetchCollections, setCollection, updateSettings ]);

  const { modalOpen, contextHolder } = useModalTrialCollection({
    onConfirm: setNewCollection,
    projectId,
  });

  const createCollectionFromFilters = useCallback(() => {
    modalOpen({ trials: { filters, sorter } });
  }, [ filters, modalOpen, sorter ]);

  const controls = (
    <div style={{ position: 'fixed', right: '30px' }}>
      <Button onClick={createCollectionFromFilters}>New Collection</Button>
      <Button onClick={saveCollection}>Save Collection</Button>
      <Select
        placeholder={collections?.length ? 'Select Collection' : 'No collections created'}
        value={settings.collection}
        onChange={(value) => setCollection(value)}>
        {[
          ...collections?.map((collection) => (
            <Select.Option key={collection.name} value={collection.name}>
              {collection.name}
            </Select.Option>
          )) ?? [],
        ]}
      </Select>
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
    setSorter,
    sorter,
  };
};
