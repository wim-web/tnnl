# 現在のリージョンを取得
data "aws_region" "current" {}

# 現在のアベイラビリティゾーンを取得
data "aws_availability_zones" "available" {
  state = "available"
}
