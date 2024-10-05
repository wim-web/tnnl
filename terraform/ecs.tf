# セキュリティグループの作成
resource "aws_security_group" "example" {
  vpc_id = aws_vpc.example.id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# ECS クラスターの作成
resource "aws_ecs_cluster" "example" {
  name = "example-cluster"
}

# ECS タスク定義
resource "aws_ecs_task_definition" "example" {
  family                   = "example-task"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = aws_iam_role.ecs_task_execution_role.arn
  task_role_arn            = aws_iam_role.ecs_task_role.arn


  container_definitions = jsonencode([{
    name      = "example-container"
    image     = "alpine:latest"
    essential = true
    command   = ["sh", "-c", "tail -f /dev/null"]
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = aws_cloudwatch_log_group.example.name
        "awslogs-region"        = data.aws_region.current.name
        "awslogs-stream-prefix" = "ecs"
      }
    }
  }])
}

# ECS サービスの作成
resource "aws_ecs_service" "example" {
  name            = "example-service"
  cluster         = aws_ecs_cluster.example.id
  task_definition = aws_ecs_task_definition.example.arn
  desired_count   = 1

  capacity_provider_strategy {
    capacity_provider = "FARGATE_SPOT"
    weight            = 1
    base              = 1
  }

  network_configuration {
    assign_public_ip = true
    subnets          = [aws_subnet.example.id]
    security_groups  = [aws_security_group.example.id]
  }

  enable_execute_command = true
}

# IAMロールの作成（必要に応じて適切なポリシーを設定）
resource "aws_iam_role" "ecs_task_execution_role" {
  name = "ecsTaskExecutionRole-example"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ecs-tasks.amazonaws.com"
      }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "ecs_task_execution_role_policy" {
  role       = aws_iam_role.ecs_task_execution_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# タスク用IAMロールの作成 (タスクがAWSリソースにアクセスするためのロール)
resource "aws_iam_role" "ecs_task_role" {
  name = "ecsTaskRole-example"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ecs-tasks.amazonaws.com"
      }
    }]
  })
}

# 必要に応じて、ここにタスクがアクセスするAWSリソースのポリシーをアタッチする
resource "aws_iam_role_policy_attachment" "ecs_task_role_policy" {
  role       = aws_iam_role.ecs_task_role.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

# CloudWatch Logs Groupの作成
resource "aws_cloudwatch_log_group" "example" {
  name              = "/ecs/example-task"
  retention_in_days = 7
}


