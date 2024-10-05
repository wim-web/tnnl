# セキュリティグループの作成 (MySQL用ポートを開放)
resource "aws_security_group" "rds_sg" {
  vpc_id = aws_vpc.example.id
  name   = "rds-sg-example"

  ingress {
    from_port   = 3306
    to_port     = 3306
    protocol    = "tcp"
    cidr_blocks = [aws_vpc.example.cidr_block] # 必要に応じて変更（例: 特定のIPアドレス）
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# RDSインスタンスの作成 (最小サイズのdb.t4g.micro)
resource "aws_db_instance" "mysql" {
  allocated_storage    = 20 # ストレージサイズ (GiB)
  engine               = "mysql"
  engine_version       = "8.0" # 必要に応じてバージョンを指定
  instance_class       = "db.t4g.micro"
  identifier           = "example"
  username             = "admin"
  password             = "your_password_here"
  parameter_group_name = "default.mysql8.0" # MySQL 8.0 用のパラメータグループ
  skip_final_snapshot  = true

  # ネットワーク設定
  vpc_security_group_ids = [aws_security_group.rds_sg.id]
  db_subnet_group_name   = aws_db_subnet_group.default.name

  # 自動バックアップの設定
  backup_retention_period = 0 # 日数を指定

  # マルチAZ配置をオフ (テスト環境や低コストにするため)
  multi_az = false

  # RDSインスタンスのパブリックアクセスを無効化
  publicly_accessible = false

  # タグの追加
  tags = {
    Name = "my-rds-instance"
  }
}

# RDS Subnet Groupの作成
resource "aws_db_subnet_group" "default" {
  subnet_ids = [aws_subnet.example.id, aws_subnet.example2.id]

  tags = {
    Name = "default"
  }
}

output "rds_endpoint" {
  value = aws_db_instance.mysql.endpoint
}
