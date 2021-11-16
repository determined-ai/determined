import { EditOutlined } from '@ant-design/icons';
import { Button, Card, Space, Tooltip } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import EditableMetadata, { Metadata } from './EditableMetadata';
import Spinner from './Spinner';

interface Props {
  forceEdit: boolean;
  metadata?: Metadata;
  onSave?: (newMetadata: Metadata) => Promise<void>;
}

const MetadataCard: React.FC<Props> = ({ metadata = {}, onSave, forceEdit }: Props) => {
  const [ isEditing, setIsEditing ] = useState(false);
  const [ isLoading, setIsLoading ] = useState(false);
  const [ editedMetadata, setEditedMetadata ] = useState<Metadata>(metadata);

  const metadataArray = useMemo(() => {
    return Object.entries(metadata).map(([ key, value ]) => {
      return ({ content: value, label: key });
    });
  }, [ metadata ]);

  const editMetadata = useCallback(() => {
    setIsEditing(true);
  }, []);

  const saveMetadata = useCallback(async () => {
    setIsEditing(false);
    setIsLoading(true);
    await onSave?.(editedMetadata);
    setIsLoading(false);
  }, [ editedMetadata, onSave ]);

  const cancelEditMetadata = useCallback(() => {
    setIsEditing(false);
  }, []);

  useEffect(() => {
    if (forceEdit) {
      setIsEditing(true);
    }
  }, [ forceEdit ]);

  if (!(metadataArray.length > 0 || isEditing || isLoading)) {
    return (<div />);
  }

  return (
    <Card
      extra={isEditing ? (
        <Space size="small">
          <Button size="small" onClick={cancelEditMetadata}>Cancel</Button>
          <Button size="small" type="primary" onClick={saveMetadata}>Save</Button>
        </Space>
      ) : (
        <Tooltip title="Edit">
          <EditOutlined onClick={editMetadata} />
        </Tooltip>
      )}
      title={'Metadata'}>
      <Spinner spinning={isLoading}>
        <EditableMetadata
          editing={isEditing}
          metadata={editedMetadata}
          updateMetadata={setEditedMetadata} />
      </Spinner>
    </Card>);
};

export default MetadataCard;
