# Takes a source 24/7 pagerduty schedule, and mirrors it to another schedule
# that is daytime hours only. You should create this new daytime schedule via
# the UI, set it to be "only between certain hours", then plug the ID into this
# script.
#
# API key file should include one line with your API key on it.
#
# Run with no arguments for usage details.
# Run unit tests by passing "test" as the only CLI argument.
require 'date'
require 'json'
require 'tempfile'
require 'pp'

def temp_with_content(data)
  tempfile = Tempfile.new('pd-data')
  tempfile.write(data.to_json)
  tempfile.flush
  tempfile
end

def json_exec(cmd)
  result = `#{cmd}`
  JSON.parse(result)
rescue JSON::ParserError
end

def curl_cmd(extra)
  return <<-EOS
    curl -s -H "Content-type: application/json" \
          -H "Authorization: Token token=#{$TOKEN}" \
    #{extra}
  EOS
end

def get(path)
  today = Date.today
  cmd = curl_cmd <<-EOS
      -X GET -G \
      --data-urlencode "since=#{today}" \
      --data-urlencode "until=#{today + 14}" \
      --data-urlencode "editable=true" \
      "https://#{$project}.pagerduty.com/api/v1#{path}"
  EOS

  json_exec(cmd)
end

def delete(path)
  cmd = curl_cmd <<-EOS
      -X DELETE "https://#{$project}.pagerduty.com/api/v1#{path}"
  EOS

  json_exec(cmd)
end

def put(path, data)
  tempfile = temp_with_content(data.to_json)

  cmd = "cat #{tempfile.path} | " + curl_cmd(<<-EOS)
     -X PUT -d @- "https://#{$project}.pagerduty.com/api/v1#{path}"
  EOS

  json_exec(cmd)
end

def post(path, data)
  cmd = curl_cmd(<<-EOS)
     -X POST -d '#{data.to_json}' "https://#{$project}.pagerduty.com/api/v1#{path}"
  EOS

  json_exec(cmd)
end

# Breaks a time range into its daytime intervals of 8am - 9pm
def daytime_intervals(s, e, day_start = 8, day_end = 21)
  def hours(x)
    x / 24.0
  end

  result = []

  # Temporary start and end variables. These are shifted around inplace.
  ts = s
  te = e

  while ts < e
    # Move ts forward in time as necessary until it is inside the time range
    unless (day_start...day_end).cover?(ts.hour)
      # Advance ts to next break
      if ts.hour < day_start
        ts += hours(day_start - ts.hour)
      else
        ts += hours(24 + day_start - ts.hour)
      end
    end

    # ts is in range now, check te
    if (day_start..day_end).cover?(te.hour) && te.to_date == ts.to_date
      # both ts and te are in range, so add the interval and reset both
      # variables
      result << [ts, te].map(&:to_s)
      ts = te
      te = e
    else
      # Modify te to be in range, the interval will be picked up on the next
      # loop assuming all variables are still valid.
      te = ts
      te += hours(day_end - ts.hour)
    end
  end

  result
end

def main(normal_schedule, daytime_schedule)
  schedule = get('/schedules/%s' % normal_schedule)
  daytime  = get('/schedules/%s' % daytime_schedule)

  restriction = daytime['schedule']['schedule_layers'][0]['restrictions'][0]

  day_start = restriction['start_time_of_day']
  duration = restriction['duration_seconds']

  unless day_start && duration
    $stderr.puts daytime.pretty_inspect
    $stderr.puts
    raise <<-EOS
      Could not locate start_time_of_day and duration in daytime schedule. Either
      it is configured incorrectly, or the JSON format is not as expected.
    EOS
  end

  day_start = day_start.split(':')[0].to_i
  day_end = day_start + duration / 60 / 60

  daytime['schedule']['schedule_layers'][0]['rendered_schedule_entries'] =
    schedule['schedule']['schedule_layers'][0]['rendered_schedule_entries']

  daytime['schedule']['schedule_layers'][0]['users'] =
    schedule['schedule']['schedule_layers'][0]['users']

  daytime['schedule'].delete('final_schedule')

  put('/schedules/%s' % daytime_schedule, daytime)

  get('/schedules/%s/overrides' % daytime_schedule)['overrides'].each do |override|
    delete('/schedules/%s/overrides/%s' % [daytime_schedule, override.fetch('id')])
  end

  schedule['schedule']['overrides_subschedule']['rendered_schedule_entries'].each do |override|
    override.delete('id')

    s = DateTime.parse(override['start'])
    e = DateTime.parse(override['end'])

    daytime_intervals(s, e, day_start, day_end).each do |(ds, de)|
      post '/schedules/%s/overrides' % daytime_schedule, {'override' => {
        'user_id' => override['user']['id'],
        'start'   => ds.to_s,
        'end'     => de.to_s
      }}
    end
  end
end

require 'minitest/unit'

class DateMathTest < MiniTest::Unit::TestCase
  def t(day, hour)
    DateTime.parse("#{day}T#{hour}:00-08:00")
  end

  def stringify(x)
    x.map {|y| y.map(&:to_s) }
  end

  def test_10am_24_hour_exception
    s = t("2013-03-05", "10:00")
    e = t("2013-03-06", "10:00")

    expected = stringify [
      [s, t("2013-03-05", "21:00")],
      [t("2013-03-06", "08:00"), e]
    ]

    assert_equal expected, daytime_intervals(s, e)
  end

  def test_2day_exception
    s = t("2013-03-05", "10:00")
    e = t("2013-03-07", "10:00")

    expected = stringify [
      [s, t("2013-03-05", "21:00")],
      [t("2013-03-06", "08:00"), t("2013-03-06", "21:00")],
      [t("2013-03-07", "08:00"), e],
    ]

    assert_equal expected, daytime_intervals(s, e)
  end

  def test_no_exception
    s = t("2013-03-05", "22:00")
    e = t("2013-03-05", "23:00")

    assert_equal [], daytime_intervals(s, e)
  end

  def test_early_start
    s = t("2013-03-05", "07:00")
    e = t("2013-03-05", "11:00")

    expected = stringify [
      [t("2013-03-05", "08:00"), e],
    ]
    assert_equal expected, daytime_intervals(s, e)
  end

  def test_late_start
    s = t("2013-03-05", "21:00")
    e = t("2013-03-06", "09:00")

    expected = stringify [
      [t("2013-03-06", "08:00"), e],
    ]
    assert_equal expected, daytime_intervals(s, e)
  end
end

if ARGV == ['test']
  require 'minitest/autorun'
elsif ARGV.length != 4
  $stderr.puts "Usage:   #{__FILE__} <project> <api-key-file> <source_schedule_id> <daytime_schedule_id>"
  $stderr.puts "Example: #{__FILE__} square /data/pagerduty-api-key P7ABC123 P7DEF456"
  exit 1
else
  $project = ARGV.shift
  $TOKEN = File.read(ARGV.shift).strip
  normal_schedule  = ARGV.shift
  daytime_schedule = ARGV.shift

  main(normal_schedule, daytime_schedule)
end
