import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Icon from 'components/kit/Icon';
import { notification } from 'components/kit/internal/dialogApi';
import { copyToClipboard } from 'components/kit/internal/functions';

import Button from './Button';
import Tooltip, { Placement } from './Tooltip';

type IconName = 'clipboard' | 'checkmark';

interface Props {
  copiedMessage?: string;
  disabled?: boolean;
  // A get content function is used to dynamically fetch content when it is needed.
  getContent: () => string;
  tooltipPlacement?: Placement;
  onCopy?: () => void;
}

export const TOOLTIP_LABEL_DEFAULT = 'Copy to clipboard';
const TOOLTIP_LABEL_SUCCESS = 'Copied!';

const ClipboardButton: React.FC<Props> = ({
  copiedMessage = TOOLTIP_LABEL_SUCCESS,
  disabled,
  getContent,
  tooltipPlacement = 'top',
  onCopy,
}) => {
  const [iconName, setIconName] = useState<IconName>('clipboard');
  const [tooltipLabel, setTooltipLabel] = useState<string | undefined>(TOOLTIP_LABEL_DEFAULT);
  const [tooltipOpen, setTooltipOpen] = useState(false);
  const buttonRef = useRef<HTMLButtonElement>(null);

  const icon = useMemo(() => <Icon name={iconName} title={TOOLTIP_LABEL_DEFAULT} />, [iconName]);

  const handleCopyToClipboard = useCallback(async () => {
    try {
      await copyToClipboard(getContent());
      setTooltipLabel(copiedMessage);
      setIconName('checkmark');
      onCopy?.();
    } catch (e) {
      setTooltipOpen(false);
      notification.error({
        description: (e as Error)?.message,
        message: 'Unable to Copy to Clipboard',
      });
    }
  }, [copiedMessage, getContent, onCopy]);

  useEffect(() => {
    const button = buttonRef.current;
    const onMouseEnter = () => {
      setTooltipLabel(TOOLTIP_LABEL_DEFAULT);
      setIconName('clipboard');
      setTooltipOpen(true);
    };
    const onMouseLeave = () => {
      setTooltipOpen(false);
      setIconName('clipboard');
    };

    button?.addEventListener('mouseenter', onMouseEnter);
    button?.addEventListener('mouseleave', onMouseLeave);

    return () => {
      button?.removeEventListener('mouseenter', onMouseEnter);
      button?.removeEventListener('mouseleave', onMouseLeave);
    };
  }, []);

  return (
    <Tooltip content={tooltipLabel} open={tooltipOpen} placement={tooltipPlacement}>
      <Button
        aria-label={tooltipLabel}
        disabled={disabled}
        icon={icon}
        ref={buttonRef}
        onClick={handleCopyToClipboard}
      />
    </Tooltip>
  );
};

export default ClipboardButton;
