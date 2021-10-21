resource "aws_cloudfront_distribution" "distribution" {
  origin {
    custom_origin_config {
      http_port              = "80"
      https_port             = "443"
      origin_protocol_policy = "http-only"
      origin_ssl_protocols   = ["TLSv1", "TLSv1.1", "TLSv1.2"]
    }

    domain_name = "${aws_s3_bucket.docs.website_endpoint}"

    origin_id = "${local.domain}"
  }

  enabled             = true
  default_root_object = "index.html"
  aliases             = ["${local.domain}"]

  default_cache_behavior {
    viewer_protocol_policy = "redirect-to-https"
    compress               = true
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]

    target_origin_id = "${local.domain}"
    min_ttl          = 0
    default_ttl      = 3600
    max_ttl          = 31536000

    forwarded_values {
      query_string = false

      cookies {
        forward = "none"
      }
    }
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn = "arn:aws:acm:us-east-1:573932760021:certificate/ccad75bf-3714-4e7e-86e5-65cb303f8fdf"
    ssl_support_method  = "sni-only"
  }
}
