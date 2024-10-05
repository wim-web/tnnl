# VPCの作成
resource "aws_vpc" "example" {
  cidr_block = "10.120.0.0/16"
}

# インターネットゲートウェイの作成
resource "aws_internet_gateway" "example" {
  vpc_id = aws_vpc.example.id
}

# パブリックサブネットの作成
resource "aws_subnet" "example" {
  vpc_id                  = aws_vpc.example.id
  cidr_block              = cidrsubnet(aws_vpc.example.cidr_block, 8, 1)
  availability_zone       = data.aws_availability_zones.available.names[0]
  map_public_ip_on_launch = true
}

resource "aws_subnet" "example2" {
  vpc_id                  = aws_vpc.example.id
  cidr_block              = cidrsubnet(aws_vpc.example.cidr_block, 8, 2)
  availability_zone       = data.aws_availability_zones.available.names[1]
  map_public_ip_on_launch = true
}

# ルートテーブルの作成
resource "aws_route_table" "example" {
  vpc_id = aws_vpc.example.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.example.id
  }
}

# サブネットにルートテーブルを関連付け
resource "aws_route_table_association" "example" {
  subnet_id      = aws_subnet.example.id
  route_table_id = aws_route_table.example.id
}
