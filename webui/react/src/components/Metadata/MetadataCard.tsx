import Button from 'hew/Button';
import Icon from 'hew/Icon';
import Row from 'hew/Row';
import Spinner from 'hew/Spinner';
import Surface from 'hew/Surface';
import { Title, TypographySize } from 'hew/Typography';
import React, { useCallback, useMemo, useState } from 'react';

import { Metadata } from 'types';
import handleError, { ErrorType } from 'utils/error';

import EditableMetadata from './EditableMetadata';

interface Props {
  disabled?: boolean;
  metadata?: Metadata;
  onSave?: (newMetadata: Metadata) => Promise<void>;
}

const MetadataCard: React.FC<Props> = ({ disabled = false, metadata = {}, onSave }: Props) => {
  const [isEditing, setIsEditing] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [editedMetadata, setEditedMetadata] = useState<Metadata>(metadata ?? {});

  const metadataArray = useMemo(() => {
    return Object.entries(metadata ?? {}).map(([key, value]) => {
      return { content: value, label: key };
    });
  }, [metadata]);

  const editMetadata = useCallback(() => {
    if (disabled) return;
    setIsEditing(true);
  }, [disabled]);

  const saveMetadata = useCallback(async () => {
    try {
      setIsLoading(true);
      await onSave?.(editedMetadata);
      setIsEditing(false);
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to update metadata.',
        silent: true,
        type: ErrorType.Api,
      });
    }
    setIsLoading(false);
  }, [editedMetadata, onSave]);

  const cancelEditMetadata = useCallback(() => {
    setEditedMetadata(metadata);
    setIsEditing(false);
  }, [setEditedMetadata, metadata]);

  const showPlaceholder = useMemo(() => {
    return metadataArray.length === 0 && !isEditing;
  }, [isEditing, metadataArray.length]);

  return (
    <Surface>
      <div>
        <Row justifyContent="space-between">
          <Title size={TypographySize.S}>Metadata</Title>
          {isEditing ? (
            <div>
              <Button size="small" onClick={cancelEditMetadata}>
                Cancel
              </Button>
              <Button size="small" type="primary" onClick={saveMetadata}>
                Save
              </Button>
            </div>
          ) : (
            disabled || (
              <Button
                icon={<Icon name="pencil" showTooltip size="small" title="Edit" />}
                type="text"
                onClick={editMetadata}
              />
            )
          )}
        </Row>
        <Surface>
          {showPlaceholder ? (
            <div
              style={{ color: 'var(--theme-colors-monochrome-9)', fontStyle: 'italic' }}
              onClick={editMetadata}>
              {disabled ? 'No metadata present.' : 'Add Metadata...'}
            </div>
          ) : (
            <Spinner spinning={isLoading}>
              <EditableMetadata
                editing={isEditing}
                metadata={editedMetadata}
                updateMetadata={setEditedMetadata}
              />
            </Spinner>
          )}
        </Surface>
      </div>
    </Surface>
  );
};

export default MetadataCard;
