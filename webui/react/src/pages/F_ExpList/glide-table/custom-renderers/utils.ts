interface CornerRadius {
  tl: number;
  tr: number;
  bl: number;
  br: number;
}

export function roundedRect(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  width: number,
  height: number,
  radius: number | CornerRadius,
): void {
  if (radius === 0) {
    ctx.rect(x, y, width, height);
    return;
  }
  if (typeof radius === 'number') {
    radius = { bl: radius, br: radius, tl: radius, tr: radius };
  }

  // restrict radius to a reasonable max
  radius = {
    bl: Math.min(radius.bl, height / 2, width / 2),
    br: Math.min(radius.br, height / 2, width / 2),
    tl: Math.min(radius.tl, height / 2, width / 2),
    tr: Math.min(radius.tr, height / 2, width / 2),
  };

  ctx.moveTo(x + radius.tl, y);
  ctx.arcTo(x + width, y, x + width, y + radius.tr, radius.tr);
  ctx.arcTo(x + width, y + height, x + width - radius.br, y + height, radius.br);
  ctx.arcTo(x, y + height, x, y + height - radius.bl, radius.bl);
  ctx.arcTo(x, y, x + radius.tl, y, radius.tl);
}

export function drawArrow(
  ctx: CanvasRenderingContext2D,
  direction: 'down' | 'up' = 'up',
  x: number,
  y: number,
  width = 8,
  height = 12,
): void {
  const headDelta = width / 2;

  ctx.beginPath();

  switch (direction) {
    case 'up':
      ctx.moveTo(x, y + headDelta);
      ctx.lineTo(x + headDelta, y);
      ctx.lineTo(x + width, y + headDelta);
      ctx.moveTo(x + headDelta, y);
      ctx.lineTo(x + headDelta, y + height);
      break;
    case 'down':
      ctx.moveTo(x, y + height - headDelta);
      ctx.lineTo(x + headDelta, y + height);
      ctx.lineTo(x + width, y + height - headDelta);
      ctx.moveTo(x + headDelta, y);
      ctx.lineTo(x + headDelta, y + height);
      break;
  }

  ctx.closePath();
  ctx.stroke();
}

function truncate(
  ctx: CanvasRenderingContext2D,
  text: string,
  x: number,
  maxWidth: number,
  suffix = '…',
): string {
  const ellipsisWidth = ctx.measureText(suffix).width;
  let newText = text;
  let textWidth = ctx.measureText(text).width;

  if (textWidth <= maxWidth || textWidth <= ellipsisWidth) {
    return text;
  } else {
    while (newText.length > 0 && textWidth + ellipsisWidth > maxWidth) {
      newText = newText.substring(0, newText.length - 1);
      textWidth = ctx.measureText(newText).width;
    }
    return newText + suffix;
  }
}
export function drawTextWithEllipsis(
  ctx: CanvasRenderingContext2D,
  text: string,
  x: number,
  y: number,
  maxWidth: number,
): void {
  const ellipsisText = truncate(ctx, text, x, maxWidth);
  ctx.fillText(ellipsisText, x, y);
}
